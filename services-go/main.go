package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/thachtb/solana-bridge/services-go/Shield"
	unshield2 "github.com/thachtb/solana-bridge/services-go/unshield"
	"math"
	"math/big"
	"strings"
)

const SHIELD = "Shield"
const UNSHIELD = "Unshield"
const INCOGNITO_PROXY = "AmEJvf5URe4MpcCKWXZSJrNuvwdKNdkt5YVDs3TUMMHB"
const PROGRAM_ID = "GqyrpvvqkcrnxF9TVhfz4CwXFTZ7QsXvjUJ5YaKTxsTH"
const VAULT_ACC = "G65gJS4feG1KXpfDXiySUGT7c6QosCJcGa4nUZsF55Du"
const SYNC_NATIVE_TAG = 0x11
const NEW_TOKEN_ACC = 0x1
const ACCCOUN_SIZE = 165

func main() {
	// init vars
	// Create a new WS client (used for confirming transactions)
	wsClient, err := ws.Connect(context.Background(), rpc.DevNet_WS)
	if err != nil {
		panic(err)
	}

	program := solana.MustPublicKeyFromBase58(PROGRAM_ID)
	incognitoProxy := solana.MustPublicKeyFromBase58(INCOGNITO_PROXY)
	vaultTokenAuthority, _, err := solana.FindProgramAddress(
		[][]byte{incognitoProxy.Bytes()},
		program,
	)
	feePayer, err := solana.PrivateKeyFromBase58("588FU4PktJWfGfxtzpAAXywSNt74AvtroVzGfKkVN1LwRuvHwKGr851uH8czM5qm4iqLbs1kKoMKtMJG4ATR7Ld2")
	if err != nil {
		panic(err)
	}
	shieldMaker, err := solana.PrivateKeyFromBase58("28BD5MCpihGHD3zUfPv4VcBizis9zxFdaj9fGJmiyyLmezT94FRd9XiiLjz5gasgyX3TmH1BU4scdVE6gzDFzrc7")
	if err != nil {
		panic(err)
	}
	vaultAcc, _, err := solana.FindProgramAddress(
		[][]byte{
			incognitoProxy.Bytes(),
			shieldMaker.PublicKey().Bytes(),
		},
		program,
	)
	if err != nil {
		panic(err)
	}
	// Create a new RPC client:
	rpcClient := rpc.New(rpc.DevNet_RPC)

	// test shield tx
	recent, err := rpcClient.GetRecentBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		panic(err)
	}
	tokenSell := solana.MustPublicKeyFromBase58("BEcGFQK1T1tSu3kvHC17cyCkQ5dvXqAJ7ExB2bb5Do7a")
	tokenBuy := solana.MustPublicKeyFromBase58("FSRvxBNrQWX2Fy2qvKMLL3ryEdRtE3PUTZBcdKwASZTU")

	fmt.Println("============ TRANSFER SOL =============")
	testPubKey, err := solana.WalletFromPrivateKeyBase58("36iz7qASXP1TECiAWJ3xESwYeRmTAF7ti3SAHaYuFJqHkmKenRACcJ9q48df1Bpn4QQSv3J89BGjuznaGAh9zPXg")
	tx4, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				1000,
				feePayer.PublicKey(),
				testPubKey.PublicKey(),
			).Build(),
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer.PublicKey()),
	)
	signers0 := []solana.PrivateKey{
		feePayer,
	}
	sig0, err := SignAndSendTx(tx4, signers0, rpcClient)
	if err != nil {
		panic(err)
	}
	spew.Dump(sig0)

	fmt.Println("============ TEST CREATE NEW ASSOCIATE TOKEN ACCOUNT INSTS =============")
	signer := solana.MustPrivateKeyFromBase58("3Y6mUgjKHdaNt5CDbRhnhzq6zwSU1CzfuLG6xaCawXQoVy4s9cpz2bhyw7ccineSCsWkFq1XPmNVSB73ZHjv85QQ")

	signerTokenAuthority, _, err := solana.FindProgramAddress(
		[][]byte{signer.PublicKey().Bytes()},
		program,
	)

	signerSellToken, _, err := solana.FindAssociatedTokenAddress(
		signerTokenAuthority,
		tokenSell,
	)

	newSignerSellTokenInst := associatedtokenaccount.NewCreateInstruction(
		feePayer.PublicKey(),
		signerTokenAuthority,
		tokenSell,
	).Build()

	signerBuyToken, _, err := solana.FindAssociatedTokenAddress(
		signerTokenAuthority,
		tokenBuy,
	)

	newSignerBuyTokenInst := associatedtokenaccount.NewCreateInstruction(
		feePayer.PublicKey(),
		signerTokenAuthority,
		tokenBuy,
	).Build()

	vaultSellToken, _, err := solana.FindAssociatedTokenAddress(
		vaultTokenAuthority,
		tokenSell,
	)

	vaultBuyToken, _, err := solana.FindAssociatedTokenAddress(
		vaultTokenAuthority,
		tokenBuy,
	)

	feePayerSellToken, _, err := solana.FindAssociatedTokenAddress(
		feePayer.PublicKey(),
		tokenSell,
	)

	newVaultSellTokenInst := associatedtokenaccount.NewCreateInstruction(
		feePayer.PublicKey(),
		vaultTokenAuthority,
		tokenSell,
	).Build()

	newVaultBuyTokenInst := associatedtokenaccount.NewCreateInstruction(
		feePayer.PublicKey(),
		vaultTokenAuthority,
		tokenBuy,
	).Build()

	fmt.Println("============ TEST SHIELD TOKEN ACCOUNT =============")
	incAddress := "12shR6fDe7ZcprYn6rjLwiLcL7oJRiek66ozzYu3B3rBxYXkqJeZYj6ZWeYy4qR4UHgaztdGYQ9TgHEueRXN7VExNRGB5t4auo3jTgXVBiLJmnTL5LzqmTXezhwmQvyrRjCbED5xW7yMMeeWarKa"
	shieldAmount := uint64(1e8)
	shieldAccounts := []*solana.AccountMeta{
		solana.NewAccountMeta(feePayerSellToken, true, false),
		solana.NewAccountMeta(vaultSellToken, true, false),
		solana.NewAccountMeta(incognitoProxy, false, false),
		solana.NewAccountMeta(feePayer.PublicKey(), false, true),
		solana.NewAccountMeta(solana.TokenProgramID, false, false),
	}
	signers := []solana.PrivateKey{
		feePayer,
	}

	shieldInstruction := shield.NewShield(
		incAddress,
		shieldAmount,
		program,
		shieldAccounts,
		byte(0),
	)
	shieldInsGenesis := shieldInstruction.Build()
	if shieldInsGenesis == nil {
		panic("Build inst got error")
	}
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			newSignerBuyTokenInst,
			newSignerSellTokenInst,
			newVaultSellTokenInst,
			newVaultBuyTokenInst,
			shieldInsGenesis,
		}[4:],
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer.PublicKey()),
	)
	if err != nil {
		panic(err)
	}
	sig, err := SignAndSendTx(tx, signers, rpcClient)
	if err != nil {
		panic(err)
	}
	spew.Dump(sig)

	fmt.Println("============ TEST UNSHIELD TOKEN ACCOUNT =============")
	txBurn2 := "860e3d8a207872c430304bdd44dfe920c62b518dff2a802e0afe04ef997f2cbd"
	tokenUnshield := solana.MustPublicKeyFromBase58("BEcGFQK1T1tSu3kvHC17cyCkQ5dvXqAJ7ExB2bb5Do7a")
	unshieldMakerRequest := solana.MustPublicKeyFromBase58("F56LmA6zUUoxQrTJ2TwTaAWAirAt4Pm93qjP8ReBgzXU")
	unshielMakerAssTokenAcc, _, err := solana.FindAssociatedTokenAddress(unshieldMakerRequest, tokenUnshield)
	if err != nil {
		panic(err)
	}

	fmt.Println(unshielMakerAssTokenAcc.String())

	vaultAssTokenAcc, _, err := solana.FindAssociatedTokenAddress(vaultTokenAuthority, tokenUnshield)
	if err != nil {
		panic(err)
	}
	fmt.Println(vaultAssTokenAcc.String())

	txburnID, err := hex.DecodeString("373f1b9482f22612c1a14ccdb6e91098b5d009bc0b97befbe856a94c8b9170ef")
	if err != nil {
		panic(err)
	}

	pdaMarkBurnID, _, err := solana.FindProgramAddress(
		[][]byte{incognitoProxy.Bytes(), txburnID},
		program,
	)

	unshieldInst := []solana.Instruction{}
	signers3 := []solana.PrivateKey{
		feePayer,
	}
	fmt.Println(tokenSell.Bytes())
	fmt.Println(signerSellToken.Bytes())

	unshieldAccounts := []*solana.AccountMeta{
		solana.NewAccountMeta(vaultAssTokenAcc, true, false),
		solana.NewAccountMeta(unshieldMakerRequest, false, false),
		solana.NewAccountMeta(vaultTokenAuthority, false, false),
		solana.NewAccountMeta(pdaMarkBurnID, true, false),
		solana.NewAccountMeta(incognitoProxy, false, false),
		solana.NewAccountMeta(solana.TokenProgramID, false, false),
		solana.NewAccountMeta(unshielMakerAssTokenAcc, true, false),
		solana.NewAccountMeta(feePayer.PublicKey(), false, true),
		solana.NewAccountMeta(solana.SystemProgramID, false, false),
	}

	_, err = rpcClient.GetAccountInfo(context.TODO(), unshielMakerAssTokenAcc)
	if err != nil {
		if err.Error() == "not found" {
			// init account
			newUnshieldTokenAcc := associatedtokenaccount.NewCreateInstruction(
				feePayer.PublicKey(),
				unshieldMakerRequest,
				tokenUnshield,
			).Build()
			unshieldInst = append(unshieldInst, newUnshieldTokenAcc)
		} else {
			panic(err)
		}
	}

	unshield := unshield2.NewUnshield(
		txBurn2,
		"getsolburnproof",
		"http://51.89.21.38:8334",
		program,
		unshieldAccounts,
	)
	unshieldInst = append(unshieldInst, unshield.Build())
	tx3, err := solana.NewTransaction(
		unshieldInst,
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer.PublicKey()),
	)
	if err != nil {
		panic(err)
	}
	sig, err = SignAndSendTx(tx3, signers3, rpcClient)
	if err != nil {
		panic(err)
	}
	spew.Dump(sig)

	fmt.Println("============ TEST SUBMIT BURN PROOF =============")
	burnProofdAccounts := []*solana.AccountMeta{
		solana.NewAccountMeta(vaultAssTokenAcc, true, false),
		solana.NewAccountMeta(signer.PublicKey(), false, false),
		solana.NewAccountMeta(vaultTokenAuthority, false, false),
		solana.NewAccountMeta(vaultAcc, true, false),
		solana.NewAccountMeta(incognitoProxy, false, false),
		solana.NewAccountMeta(solana.TokenProgramID, false, false),
		solana.NewAccountMeta(signerSellToken, true, false),
	}
	fmt.Println(burnProofdAccounts)

	fmt.Println("============ GET RAYDIUM SWAP RATE =============")
	ammAcc := solana.MustPublicKeyFromBase58("HeD1cekRWUNR25dcvW8c9bAHeKbr1r7qKEhv7pEegr4f")
	poolTokenAmm := solana.MustPublicKeyFromBase58("3qbeXHwh9Sz4zabJxbxvYGJc57DZHrFgYMCWnaeNJENT")
	pcTokenAmm := solana.MustPublicKeyFromBase58("FrGPG5D4JZVF5ger7xSChFVFL8M9kACJckzyCz8tVowz")
	swapAmount := uint64(1e12)
	poolTokenBal, err := rpcClient.GetTokenAccountBalance(context.Background(), poolTokenAmm, rpc.CommitmentConfirmed)
	if err != nil {
		panic(err)
	}
	pcTokenBal, err := rpcClient.GetTokenAccountBalance(context.Background(), pcTokenAmm, rpc.CommitmentConfirmed)
	if err != nil {
		panic(err)
	}

	fmt.Printf("pool token bal: %v \n", poolTokenBal.Value.Amount)
	fmt.Printf("pc token bal: %v \n", pcTokenBal.Value.Amount)

	ammId := solana.MustPublicKeyFromBase58("HeD1cekRWUNR25dcvW8c9bAHeKbr1r7qKEhv7pEegr4f")
	resp, err := rpcClient.GetAccountInfo(
		context.TODO(),
		ammId,
	)
	if err != nil {
		panic(err)
	}
	ammData := resp.Value.Data.GetBinary()
	swap_fee_numerator := ammData[22*8 : 23*8]
	swap_fee_denominator := ammData[23*8 : 24*8]
	swapFeeNum := binary.LittleEndian.Uint64(swap_fee_numerator)
	swapFeeDe := binary.LittleEndian.Uint64(swap_fee_denominator)
	// hardcode fee 0.3%
	fromAmountWithFee := new(big.Int).SetUint64(swapAmount * (swapFeeDe - swapFeeNum) / swapFeeDe)
	poolAmountBig, _ := new(big.Int).SetString(poolTokenBal.Value.Amount, 10)
	pcAmountBig, _ := new(big.Int).SetString(pcTokenBal.Value.Amount, 10)
	denominator := big.NewInt(0).Add(fromAmountWithFee, poolAmountBig)
	temp := big.NewInt(0).Mul(fromAmountWithFee, pcAmountBig)
	amountOut := big.NewInt(0).Div(temp, denominator).Uint64()
	fmt.Printf("Amount out: %v \n", amountOut)
	priceBefore := float64(pcAmountBig.Uint64()) / float64(poolAmountBig.Uint64())
	priceAfter := float64(pcAmountBig.Uint64()-amountOut) / float64(denominator.Uint64())
	priceImpact := math.Abs(priceBefore-priceAfter) * 100 / priceBefore
	fmt.Printf("price Impact: %v%%\n", priceImpact)

	fmt.Println("============ SWAP ON RAYDIUM =============")
	// create and mint token
	// get token amount out
	// assume amount out 0
	// swap token
	ammProgramId := solana.MustPublicKeyFromBase58("9rpQHSyFVM1dkkHFQ2TtTzPEW7DVmEyPmN8wVniqJtuC")
	swapAccounts := []*solana.AccountMeta{
		solana.NewAccountMeta(signer.PublicKey(), false, true),
		solana.NewAccountMeta(solana.TokenProgramID, false, false),
		solana.NewAccountMeta(ammAcc, true, false),                                                                          // amm account
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("DhVpojXMTbZMuTaCgiiaFU7U8GvEEhnYo4G9BUdiEYGh"), false, false), // amm authority
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("HboQAt9BXyejnh6SzdDNTx4WELMtRRPCr7pRSLpAW7Eq"), true, false),  // amm open orders
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("6TzAjFPVZVMjbET8vUSk35J9U2dEWFCrnbHogsejRE5h"), true, false),  // amm target order
		solana.NewAccountMeta(poolTokenAmm, true, false),                                                                    // pool_token_coin Amm
		solana.NewAccountMeta(pcTokenAmm, true, false),                                                                      // pool_token_pc Amm
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("DESVgJVGajEgKGXhb6XmqDHGz3VjdgP7rEVESBgxmroY"), false, false), // serum dex
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("3tsrPhKrWHWMB8RiPaqNxJ8GnBhZnDqL4wcu5EAMFeBe"), true, false),  // serum market accounts
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("ANHHchetdZVZBuwKWgz8RSfVgCDsRpW9i2BNWrmG9Jh9"), true, false),  // bid account
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("ESSri17GNbVttqrp7hrjuXtxuTcCqytnrMkEqr29gMGr"), true, false),  // ask account
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("FGAW7QqNJGFyhakh5jPzGowSb8UqcSJ95ZmySeBgmVwt"), true, false),  // event q accounts
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("E1E5kQqWXkXbaqVzpY5P2EQUSi8PNAHdCnqsj3mPWSjG"), true, false),  // coin vault
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("3sj6Dsw8fr8MseXpCnvuCSczR8mQjCWNyWDC5cAfEuTq"), true, false),  // pc vault
		solana.NewAccountMeta(solana.MustPublicKeyFromBase58("C2fDkZJqHH5PXyQ7UWBNZsmu6vDXxrEbb9Ex9KF7XsAE"), false, false), // vault signer account
		solana.NewAccountMeta(signerSellToken, true, false),                                                                 // source token acc
		solana.NewAccountMeta(signerBuyToken, true, false),                                                                  // dest token acc
		solana.NewAccountMeta(signerTokenAuthority, false, false),                                                           // user owner
		solana.NewAccountMeta(ammProgramId, false, false),                                                                   // user owner
	}

	// swapbasein
	tag := byte(9)

	amountInBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountInBytes, swapAmount)

	amountOutBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountOutBytes, amountOut)
	swapData := append([]byte{tag}, amountInBytes...)
	swapData = append(swapData, amountOutBytes...)
	data := append([]byte{0x3}, []byte{byte(len(swapData))}...)
	data = append(data, swapData...)
	data = append(data, []byte{byte(len(swapAccounts) - 2)}...)
	data = append(data, []byte{byte(17)}...)
	fmt.Printf("data %v\n", data)
	txSwap, err := solana.NewTransaction(
		[]solana.Instruction{
			solana.NewInstruction(
				program,
				swapAccounts,
				data,
			),
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer.PublicKey()),
	)
	if err != nil {
		panic(err)
	}

	signersSwap := []solana.PrivateKey{
		feePayer,
		signer,
	}

	txSig, err := SignAndSendTx(txSwap, signersSwap, rpcClient)
	if err != nil {
		//panic(err)
	}
	spew.Dump(txSig)

	fmt.Println("==============================================================================")
	fmt.Println("==============================================================================")
	fmt.Println("=============================== REQUEST WITHDRAW =============================")
	fmt.Println("==============================================================================")

	shieldAmount2 := uint64(1e5)
	shieldAccounts2 := []*solana.AccountMeta{
		solana.NewAccountMeta(signerBuyToken, true, false),
		solana.NewAccountMeta(vaultBuyToken, true, false),
		solana.NewAccountMeta(incognitoProxy, false, false),
		solana.NewAccountMeta(signer.PublicKey(), false, true),
		solana.NewAccountMeta(signerTokenAuthority, false, false),
		solana.NewAccountMeta(solana.TokenProgramID, false, false),
	}
	signersShield := []solana.PrivateKey{
		feePayer,
		signer,
	}

	shieldInstruction2 := shield.NewShield(
		incAddress,
		shieldAmount2,
		program,
		shieldAccounts2,
		byte(4),
	)
	shieldInsGenesis2 := shieldInstruction2.Build()
	if shieldInsGenesis == nil {
		panic("Build inst got error")
	}
	tx, err = solana.NewTransaction(
		[]solana.Instruction{
			shieldInsGenesis2,
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer.PublicKey()),
	)
	if err != nil {
		panic(err)
	}
	sig, err = SignAndSendTx(tx, signersShield, rpcClient)
	if err != nil {
		panic(err)
	}
	spew.Dump(sig)

	fmt.Println("==============================================================================")
	fmt.Println("==============================================================================")
	fmt.Println("====================== GET TX INFO IN TRANSFER TX ============================")
	fmt.Println("==============================================================================")

	// catch
	txhash, err := solana.SignatureFromBase58("2yZtM9jcYG1jZzRLWqgLAnG6tqA7n5fQm8f5Ki1xmwHxaX5vnPK1tbeyarmpw2TYxRUYuJ8C5v1fo8ey5H4Byb9t")
	if err != nil {
		panic(err)
	}
	opts := rpc.GetTransactionOpts{
		Commitment: rpc.CommitmentConfirmed,
		Encoding:   solana.EncodingBase58,
	}
	txInfo, err := rpcClient.GetTransaction(context.Background(), txhash, &opts)
	if err != nil {
		panic(err)
	}
	fmt.Printf("account pre balances info %+v\n", txInfo.Meta.PostBalances)
	fmt.Printf("account balances info %+v\n", txInfo.Meta.PostBalances)
	fmt.Printf("token account balances info %+v\n", txInfo.Meta.PostTokenBalances)
	txDecode, err := solana.TransactionFromDecoder(bin.NewBinDecoder(txInfo.Transaction.GetBinary()))
	if err != nil {
		panic(err)
	}
	fmt.Printf("key list %+v \n", txDecode.Message.AccountKeys)

	fmt.Println("==============================================================================")
	fmt.Println("==============================================================================")
	fmt.Println("============ TEST FULL FLOW SHIELD AND UNSHIELD FOR NATIVE TOKEN =============")
	fmt.Println("==============================================================================")
	solAmount := uint64(1e7)
	txBurn := "33cdf06f7dacb8ce3feccedff3cda852f0013bbf9480a69a3a473861081fd01c"
	recent, err = rpcClient.GetRecentBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		panic(err)
	}
	shieldNativeTokenAcc, _, err := solana.FindAssociatedTokenAddress(
		shieldMaker.PublicKey(),
		solana.SolMint,
	)
	if err != nil {
		panic(err)
	}

	// check account exist
	_, err = rpcClient.GetAccountInfo(context.TODO(), solana.MustPublicKeyFromBase58("D1ts5r3KWzeK16cm5Qqx9Rq8cbGmc9MmvgJzY3ZdV7Si"))
	if err != nil {
		if err.Error() == "not found" {
			// init account
			fmt.Println("do init account")
		} else {
			panic(err)
		}
	}

	fmt.Println(shieldNativeTokenAcc.String())

	// create new token account Token11..112 to shield
	shieldNativeTokenAccInst := associatedtokenaccount.NewCreateInstruction(
		feePayer.PublicKey(),
		shieldMaker.PublicKey(),
		solana.SolMint,
	).Build()

	vaultNativeTokenAcc, _, err := solana.FindAssociatedTokenAddress(
		vaultTokenAuthority,
		solana.SolMint,
	)

	if err != nil {
		panic(err)
	}
	fmt.Println(vaultNativeTokenAcc.String())

	vaultNativeTokenAccInst := associatedtokenaccount.NewCreateInstruction(
		feePayer.PublicKey(),
		vaultTokenAuthority,
		solana.SolMint,
	).Build()

	incAddress4 := "12shR6fDe7ZcprYn6rjLwiLcL7oJRiek66ozzYu3B3rBxYXkqJeZYj6ZWeYy4qR4UHgaztdGYQ9TgHEueRXN7VExNRGB5t4auo3jTgXVBiLJmnTL5LzqmTXezhwmQvyrRjCbED5xW7yMMeeWarKa"
	shieldAccounts4 := []*solana.AccountMeta{
		solana.NewAccountMeta(shieldNativeTokenAcc, true, false),
		solana.NewAccountMeta(vaultNativeTokenAcc, true, false),
		solana.NewAccountMeta(incognitoProxy, false, false),
		solana.NewAccountMeta(shieldMaker.PublicKey(), false, true),
		solana.NewAccountMeta(solana.TokenProgramID, false, false),
	}
	signers4 := []solana.PrivateKey{
		feePayer,
		shieldMaker,
	}

	shieldInstruction4 := shield.NewShield(
		incAddress4,
		solAmount,
		program,
		shieldAccounts4,
		byte(0),
	)

	// build sync native token program
	syncNativeeInst := solana.NewInstruction(
		solana.TokenProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(shieldNativeTokenAcc, true, false),
		},
		[]byte{SYNC_NATIVE_TAG},
	)

	tx4, err = solana.NewTransaction(
		[]solana.Instruction{
			shieldNativeTokenAccInst,
			vaultNativeTokenAccInst,
			system.NewTransferInstruction(
				solAmount,
				shieldMaker.PublicKey(),
				shieldNativeTokenAcc,
			).Build(),
			syncNativeeInst,
			shieldInstruction4.Build(),
		}[2:],
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer.PublicKey()),
	)
	if err != nil {
		panic(err)
	}
	sig4, err := SignAndSendTx(tx4, signers4, rpcClient)
	if err != nil {
		panic(err)
	}
	spew.Dump(sig4)

	// unshield sol
	nativeAccountToken, err := solana.WalletFromPrivateKeyBase58("YpRLgTL3DPc83MTjdsVE6ALv5RqUQt3jZ35aVQDxeAmJLqsDhXZFtPnFXganq6DfQ7Q91guGQjKc13YMVjyX8vP")
	if err != nil {
		panic(err)
	}

	unshieldAccounts5 := []*solana.AccountMeta{
		solana.NewAccountMeta(vaultNativeTokenAcc, true, false),
		solana.NewAccountMeta(testPubKey.PublicKey(), true, false),
		solana.NewAccountMeta(vaultTokenAuthority, false, false),
		solana.NewAccountMeta(vaultAcc, true, false),
		solana.NewAccountMeta(incognitoProxy, false, false),
		solana.NewAccountMeta(solana.TokenProgramID, false, false),
		solana.NewAccountMeta(nativeAccountToken.PublicKey(), true, false),
	}

	signers5 := []solana.PrivateKey{
		feePayer,
		nativeAccountToken.PrivateKey,
	}

	unshield5 := unshield2.NewUnshield(
		txBurn,
		"getsolburnproof",
		"http://51.89.21.38:8334",
		program,
		unshieldAccounts5,
	)
	exemptLamport, err := rpcClient.GetMinimumBalanceForRentExemption(context.Background(), ACCCOUN_SIZE, rpc.CommitmentConfirmed)
	if err != nil {
		panic(err)
	}

	// build create new token acc
	newAccToken := solana.NewInstruction(
		solana.TokenProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(nativeAccountToken.PublicKey(), true, false),
			solana.NewAccountMeta(solana.SolMint, false, false),
			solana.NewAccountMeta(vaultTokenAuthority, false, false),
			solana.NewAccountMeta(solana.SysVarRentPubkey, false, false),
		},
		[]byte{NEW_TOKEN_ACC},
	)

	tx5, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewCreateAccountInstruction(
				exemptLamport,
				ACCCOUN_SIZE,
				solana.TokenProgramID,
				feePayer.PublicKey(),
				nativeAccountToken.PublicKey(),
			).Build(),
			newAccToken,
			unshield5.Build(),
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer.PublicKey()),
	)
	sig, err = SignAndSendTx(tx5, signers5, rpcClient)
	if err != nil {
		panic(err)
	}
	spew.Dump(sig)

	fmt.Println("============ TEST LISTEN TRANSFER TOKEN EVENT =============")
	//wsClient.ProgramSubscribeWithOpts(
	//	solana.SystemProgramID,
	//	rpc.CommitmentFinalized,
	//	solana.EncodingJSONParsed,
	//	[]rpc.RPCFilter{},
	//)

	fmt.Println("============ TEST LISTEN TRANSFER SOL EVENT =============")
	//wsClient.ProgramSubscribeWithOpts(
	//	solana.TokenProgramID,
	//	rpc.CommitmentFinalized,
	//	solana.EncodingJSONParsed,
	//	[]rpc.RPCFilter{},
	//)

	fmt.Println("============ TEST LISTEN SHIELD EVENT =============")
	// listen shield to vault logs
	{
		// Subscribe to log events that mention the provided pubkey:
		sub, err := wsClient.LogsSubscribeMentions(
			program,
			rpc.CommitmentFinalized,
		)
		if err != nil {
			panic(err)
		}
		defer sub.Unsubscribe()

		for {
			got, err := sub.Recv()
			if err != nil {
				panic(err)
			}
			// dump to struct { signature , error, value }
			spew.Dump(got)
			processShield(got)
		}
	}
}

func processShield(logs *ws.LogResult) {
	if logs.Value.Err != nil {
		fmt.Printf("the transaction failed %v \n", logs.Value.Err)
		return
	}

	if len(logs.Value.Logs) < 7 {
		fmt.Printf("invalid shield logs, length must greate than 7 %v \n", logs.Value.Err)
		return
	}
	// todo: check signature and store if new
	//logs.Value.Signature

	shieldLogs := logs.Value.Logs
	// check shield instruction
	if !strings.Contains(shieldLogs[1], SHIELD) {
		fmt.Printf("invalid instruction %s\n", shieldLogs[1])
		return
	}

	shieldInfoSplit := strings.Split(shieldLogs[6], ":")
	if len(shieldInfoSplit) < 3 {
		fmt.Printf("invalid shield logs %+v\n", logs)
		return
	}

	shieldInfo := strings.Split(shieldInfoSplit[2], ",")
	if len(shieldInfo) < 4 {
		fmt.Printf("invalid shield info %v\n", shieldInfo)
		return
	}

	incognitoProxy := shieldInfo[0]
	if incognitoProxy != INCOGNITO_PROXY {
		fmt.Printf("invalid incognito proxy %v \n", incognitoProxy)
		return
	}
	incAddress := shieldInfo[1]
	tokenID := shieldInfo[2]
	amount := shieldInfo[3]

	fmt.Printf("shield with inc address %s token id %s and amount %s \n", incAddress, tokenID, amount)
}

func SignAndSendTx(tx *solana.Transaction, signers []solana.PrivateKey, rpcClient *rpc.Client) (solana.Signature, error) {
	_, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		for _, candidate := range signers {
			if candidate.PublicKey().Equals(key) {
				return &candidate
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("unable to sign transaction: %v \n", err)
		return solana.Signature{}, err
	}
	// send tx
	signature, err := rpcClient.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Printf("unable to send transaction: %v \n", err)
		return solana.Signature{}, err
	}
	return signature, nil
}
