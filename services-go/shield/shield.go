package shield

import (
	"encoding/binary"
	"fmt"
	"github.com/gagliardetto/solana-go"
)

const SHIELD_TAG = 0x00
const WITHDRAW_REQUEST = 0x05

type Shield struct {
	incAddress string
	amount     uint64
	programID  solana.PublicKey
	accounts   []*solana.AccountMeta
	tag        byte
}

func NewShield(incAddress string, amount uint64, programID solana.PublicKey, accounts []*solana.AccountMeta, tag byte) Shield {
	return Shield{
		incAddress: incAddress,
		amount:     amount,
		programID:  programID,
		accounts:   accounts,
		tag:        tag,
	}
}

func (sh *Shield) Build() *solana.GenericInstruction {
	if len(sh.incAddress) != 148 {
		fmt.Printf("invalid inc address %v \n", sh.incAddress)
		return nil
	}
	if len(sh.accounts) == 0 {
		fmt.Printf("invalid account list %v \n", sh.incAddress)
		return nil
	}

	// build instruction
	// convert string to byte array
	incAddressBytes := []byte(sh.incAddress)
	// convert amount to bid le endian representation 8 bytes
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, sh.amount)
	// instruction tag + shield amount + inc
	temp1 := append([]byte{sh.tag}, amountBytes...)
	shieldData := append(temp1, incAddressBytes...)
	accountSlice := solana.AccountMetaSlice{}
	err := accountSlice.SetAccounts(
		sh.accounts,
	)
	if err != nil {
		fmt.Printf("init account slice failed %v \n", err)
		return nil
	}

	return solana.NewInstruction(
		sh.programID,
		accountSlice,
		shieldData,
	)
}
