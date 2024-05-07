package unshield

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
)

const UNSHIELD_TAG = 0x1

type decodedProof struct {
	Instruction []byte
	Heights     [2]*big.Int

	InstPaths       [2][][32]byte
	InstPathIsLefts [2][]bool
	InstRoots       [2][32]byte
	BlkData         [2][32]byte
	SigIdxs         [2][]*big.Int
	Sigs            [2][]string
}

type getProofResult struct {
	Result GetInstructionProof
	Error  struct {
		Code       int
		Message    string
		StackTrace string
	}
}

type Unshield struct {
	burnTx         string
	getProofMethod string
	incFullNode    string
	programID      solana.PublicKey
	accounts       []*solana.AccountMeta
}

func NewUnshield(burnTx string, getProofMethod string, incFullNode string, programID solana.PublicKey, accounts []*solana.AccountMeta) Unshield {
	return Unshield{
		burnTx,
		getProofMethod,
		incFullNode,
		programID,
		accounts,
	}
}

func (us *Unshield) Build() *solana.GenericInstruction {
	proof, err := getAndDecodeBurnProof(us.incFullNode, us.burnTx, us.getProofMethod)
	if err != nil {
		fmt.Printf("can not get and decode proof %v", err)
		return nil
	}

	// build unshield instruction
	tag := UNSHIELD_TAG
	//if us.getProofMethod != "getsolburnproof" {
	//	tag = SUBMIT_PROOF
	//}
	temp := append([]byte{byte(tag)}, proof.Instruction...)
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, proof.Heights[0].Uint64())
	temp = append(temp, heightBytes...)

	instPathLength := len(proof.InstPaths[0])
	temp = append(temp, []byte{byte(instPathLength)}...)
	for _, v := range proof.InstPaths[0] {
		temp = append(temp, v[:]...)
	}

	instPathLeftLength := len(proof.InstPathIsLefts[0])
	temp = append(temp, []byte{byte(instPathLeftLength)}...)
	for _, v := range proof.InstPathIsLefts[0] {
		temp4 := byte(0x0)
		if v {
			temp4 = byte(0x1)
		}
		temp = append(temp, []byte{temp4}...)
	}

	temp = append(temp, proof.InstRoots[0][:]...)
	temp = append(temp, proof.BlkData[0][:]...)

	indexLength := len(proof.SigIdxs[0])
	temp = append(temp, []byte{byte(indexLength)}...)
	for _, v := range proof.SigIdxs[0] {
		index := byte(v.Uint64())
		temp = append(temp, []byte{index}...)
	}

	signaturesLength := len(proof.Sigs[0])
	temp = append(temp, []byte{byte(signaturesLength)}...)
	for _, v := range proof.Sigs[0] {
		sig, err := hex.DecodeString(v)
		if err != nil {
			fmt.Printf("invalid beacon signature %v", err)
			return nil
		}
		temp = append(temp, sig...)
	}

	accountSlice := solana.AccountMetaSlice{}
	err = accountSlice.SetAccounts(
		us.accounts,
	)
	if err != nil {
		fmt.Printf("init account slice failed %v \n", err)
		return nil
	}

	return solana.NewInstruction(
		us.programID,
		accountSlice,
		temp,
	)
}

func getAndDecodeBurnProof(
	incBridgeHost string,
	txID string,
	rpcMethod string,
) (*decodedProof, error) {
	body, err := getBurnProofV2(incBridgeHost, txID, rpcMethod)
	if err != nil {
		return nil, err
	}
	if len(body) < 1 {
		return nil, fmt.Errorf("burn proof for deposit to SC not found")
	}

	r := getProofResult{}
	err = json.Unmarshal([]byte(body), &r)
	if err != nil {
		return nil, err
	}
	return decodeProof(&r)
}

func getBurnProofV2(
	incBridgeHost string,
	txID string,
	rpcMethod string,
) (string, error) {
	if len(txID) == 0 {
		return "", errors.New("invalid tx burn id ")
	}
	payload := strings.NewReader(fmt.Sprintf("{\n    \"id\": 1,\n    \"jsonrpc\": \"1.0\",\n    \"method\": \"%s\",\n    \"params\": [\n    \t\"%s\"\n    ]\n}", rpcMethod, txID))

	req, _ := http.NewRequest("POST", incBridgeHost, payload)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func decodeProof(r *getProofResult) (*decodedProof, error) {
	//r.Result.Instruction = "9e0198114aab002770023fa97352bfd1404a5c22f46d3b54b57acc9fb1b4a9051e55d10ffa9da8016954409eca41dc51ef8fe9f2303af250c33773ee531f3b01bc430000000000000000000000000000000000000000000000000000000005f5e100373f1b9482f22612c1a14ccdb6e91098b5d009bc0b97befbe856a94c8b9170ed0000000000000000000000000000000000000000000000000000000000000000"
	inst := decode(r.Result.Instruction)
	fmt.Printf("inst: %d %x\n", len(inst), inst)
	fmt.Printf("instHash (isWithdrawed, without height): %x\n", keccak256(inst))

	// Block heights
	beaconHeight := big.NewInt(0).SetBytes(decode(r.Result.BeaconHeight))
	bridgeHeight := big.NewInt(0).SetBytes(decode(r.Result.BridgeHeight))
	heights := [2]*big.Int{beaconHeight, bridgeHeight}
	fmt.Printf("beaconHeight: %d\n", beaconHeight)
	fmt.Printf("bridgeHeight: %d\n", bridgeHeight)

	beaconInstRoot := decode32(r.Result.BeaconInstRoot)
	beaconInstPath := make([][32]byte, len(r.Result.BeaconInstPath))
	beaconInstPathIsLeft := make([]bool, len(r.Result.BeaconInstPath))
	for i, path := range r.Result.BeaconInstPath {
		beaconInstPath[i] = decode32(path)
		beaconInstPathIsLeft[i] = r.Result.BeaconInstPathIsLeft[i]
	}
	// fmt.Printf("beaconInstRoot: %x\n", beaconInstRoot)

	beaconBlkData := toByte32(decode(r.Result.BeaconBlkData))
	fmt.Printf("data: %s %s\n", r.Result.BeaconBlkData, r.Result.BeaconInstRoot)
	fmt.Printf("expected beaconBlkHash: %x\n", keccak256(beaconBlkData[:], beaconInstRoot[:]))

	beaconSigIdxs := []*big.Int{}
	for _, sIdx := range r.Result.BeaconSigIdxs {
		beaconSigIdxs = append(beaconSigIdxs, big.NewInt(int64(sIdx)))
	}

	// For bridge
	bridgeInstRoot := decode32(r.Result.BridgeInstRoot)
	bridgeInstPath := make([][32]byte, len(r.Result.BridgeInstPath))
	bridgeInstPathIsLeft := make([]bool, len(r.Result.BridgeInstPath))
	for i, path := range r.Result.BridgeInstPath {
		bridgeInstPath[i] = decode32(path)
		bridgeInstPathIsLeft[i] = r.Result.BridgeInstPathIsLeft[i]
	}
	// fmt.Printf("bridgeInstRoot: %x\n", bridgeInstRoot)
	bridgeBlkData := toByte32(decode(r.Result.BridgeBlkData))
	bridgeSigIdxs := []*big.Int{}
	for _, sIdx := range r.Result.BridgeSigIdxs {
		bridgeSigIdxs = append(bridgeSigIdxs, big.NewInt(int64(sIdx)))
		// fmt.Printf("bridgeSigIdxs[%d]: %d\n", i, j)
	}

	// Merge beacon and bridge proof
	instPaths := [2][][32]byte{beaconInstPath, bridgeInstPath}
	instPathIsLefts := [2][]bool{beaconInstPathIsLeft, bridgeInstPathIsLeft}
	instRoots := [2][32]byte{beaconInstRoot, bridgeInstRoot}
	blkData := [2][32]byte{beaconBlkData, bridgeBlkData}
	sigIdxs := [2][]*big.Int{beaconSigIdxs, bridgeSigIdxs}
	Sigs := [2][]string{r.Result.BeaconSigs, r.Result.BridgeSigs}

	return &decodedProof{
		Instruction:     inst,
		Heights:         heights,
		InstPaths:       instPaths,
		InstPathIsLefts: instPathIsLefts,
		InstRoots:       instRoots,
		BlkData:         blkData,
		SigIdxs:         sigIdxs,
		Sigs:            Sigs,
	}, nil
}
