package precompiles

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"

	"github.com/sirupsen/logrus"

	tfhe "github.com/fhenixprotocol/go-tfhe"
)

var logger *logrus.Logger

func InitLogger() {
	logger = newLogger()
	tfhe.InitLogger(getDefaultLogLevel())
}

func InitTfheConfig(tfheConfig *tfhe.Config) error {
	err := tfhe.InitTfhe(tfheConfig)
	if err != nil {
		logger.Error("Failed to init tfhe config with error: ", err)
		return err
	}

	logger.Info("Successfully initialized tfhe config to be: ", tfheConfig)

	return nil
}

func shouldPrintPrecompileInfo(tp *TxParams) bool {
	return tp.Commit && !tp.GasEstimation
}

func isTx(tp *TxParams) bool {
	return !tp.EthCall || tp.GasEstimation
}

// ============================
func Add(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "add"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName, " inputs not verified ", " err ", err, " input ", hex.EncodeToString(input))
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Add(rhs)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	resultHash := result.Hash()
	logger.Debug(functionName, " success ", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", resultHash.Hex())
	return resultHash[:], nil
}

func Verify(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "verify"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	if len(input) <= 1 {
		msg := functionName + " RequiredGas() input needs to contain a ciphertext and one byte for its type"
		logger.Error(msg, " len ", len(input))
		return nil, vm.ErrExecutionReverted
	}

	ctBytes := input[:len(input)-1]
	ctType := tfhe.UintType(input[len(input)-1])

	ct, err := tfhe.NewCipherTextFromBytes(ctBytes, ctType, true /* TODO: not sure + shouldn't be hardcoded */)
	if err != nil {
		logger.Error(functionName, " failed to deserialize input ciphertext",
			" err ", err,
			" len ", len(ctBytes),
			" ctBytes64 ", hex.EncodeToString(ctBytes[:minInt(len(ctBytes), 64)]))
		return nil, vm.ErrExecutionReverted
	}

	ctHash := ct.Hash()
	err = importCiphertext(state, ct, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if tp.Commit {
		logger.Debug(functionName, " success ",
			" ctHash ", ctHash.Hex(),
			" ctBytes64 ", hex.EncodeToString(ctBytes[:minInt(len(ctBytes), 64)]))
	}
	return ctHash[:], nil
}

func SealOutput(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: bool math
	functionName := "sealOutput"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	if len(input) != 64 {
		msg := functionName + " input len must be 64 bytes"
		logger.Error(msg, " input ", hex.EncodeToString(input), " len ", len(input))
		return nil, vm.ErrExecutionReverted
	}

	ct := getCiphertext(state, tfhe.BytesToHash(input[0:32]), isTx)
	if ct == nil {
		msg := "sealOutput unverified ciphertext handle"
		logger.Error(msg, " input ", hex.EncodeToString(input))
		return nil, vm.ErrExecutionReverted
	}

	if tp.GasEstimation {
		return []byte{1}, nil
	}

	decryptedValue, err := tfhe.Decrypt(*ct)
	if err != nil {
		logger.Error("failed decrypting ciphertext ", "error ", err)
		return nil, vm.ErrExecutionReverted
	}

	// Cast decrypted value to big.Int
	bgDecrypted := new(big.Int).SetUint64(decryptedValue)
	pubKey := input[32:64]
	reencryptedValue, err := encryptToUserKey(bgDecrypted, pubKey)
	if err != nil {
		logger.Error(functionName, " failed to encrypt to user key", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	logger.Debug(functionName, " success", " input ", hex.EncodeToString(input))

	return reencryptedValue, nil
}

func Decrypt(input []byte, tp *TxParams, state *FheosState) (*big.Int, error) {
	//solgen: output plaintext
	functionName := "decrypt"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	if len(input) != 32 {
		msg := functionName + " input len must be 32 bytes"
		logger.Error(msg, " input ", hex.EncodeToString(input), " len ", len(input))
		return nil, vm.ErrExecutionReverted
	}

	ct := getCiphertext(state, tfhe.BytesToHash(input[0:32]), isTx)
	if ct == nil {
		msg := functionName + " unverified ciphertext handle"
		logger.Error(msg, " input ", hex.EncodeToString(input))
		return nil, vm.ErrExecutionReverted
	}

	if tp.GasEstimation {
		return new(big.Int).SetUint64(1), nil
	}

	decryptedValue, err := tfhe.Decrypt(*ct)
	if err != nil {
		logger.Error("failed decrypting ciphertext", " error ", err)
		return nil, vm.ErrExecutionReverted
	}

	bgDecrypted := new(big.Int).SetUint64(decryptedValue)

	logger.Debug(functionName, " success", " input ", hex.EncodeToString(input))
	return bgDecrypted, nil

}

func Lte(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: return ebool
	functionName := "lte"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName, " inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)

	}

	result, err := lhs.Lte(rhs)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	resultHash := result.Hash()
	logger.Debug(functionName, " success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", resultHash.Hex())
	return resultHash[:], nil
}

func Sub(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "sub"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName, " inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// // If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Sub(rhs)
	if err != nil {
		logger.Error(functionName, " failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	resultHash := result.Hash()
	logger.Debug(functionName, " success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", resultHash.Hex())
	return resultHash[:], nil
}

func Mul(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "mul"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName, " inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Mul(rhs)
	if err != nil {
		logger.Error(functionName, " failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	return ctHash[:], nil
}

func Lt(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: return ebool
	functionName := "lt"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName, " inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Lt(rhs)
	if err != nil {
		logger.Error(functionName, " failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	resultHash := result.Hash()
	logger.Debug(functionName+" success ", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", resultHash.Hex())
	return resultHash[:], nil
}

func Select(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "select"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	control, ifTrue, ifFalse, err := get3VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName, " inputs not verified input len: ", len(input), " err: ", err)
		return nil, vm.ErrExecutionReverted
	}

	if ifTrue.UintType != ifFalse.UintType {
		msg := functionName + " operands type mismatch"
		logger.Error(msg, " ifTrue ", ifTrue.UintType, " ifFalse ", ifFalse.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, ifTrue.UintType)
	}

	result, err := control.Cmux(ifTrue, ifFalse)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	resultHash := result.Hash()
	logger.Debug(functionName, " success ", " control ", control.Hash().Hex(), " ifTrue ", ifTrue.Hash().Hex(), " ifFalse ", ifTrue.Hash().Hex(), " result ", resultHash.Hex())
	return resultHash[:], nil
}

func Req(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: input encrypted
	//solgen: return none
	functionName := "require"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	if len(input) != 32 {
		msg := functionName + " input len must be 32 bytes"
		logger.Error(msg, " input ", hex.EncodeToString(input), " len ", len(input))
		return nil, vm.ErrExecutionReverted
	}

	ct := getCiphertext(state, tfhe.BytesToHash(input), isTx)
	if ct == nil {
		msg := functionName + " unverified handle"
		logger.Error(msg, " input ", hex.EncodeToString(input))
		return nil, vm.ErrExecutionReverted
	}
	// If we are not committing to state, assume the require is true, avoiding any side effects
	// (i.e. mutatiting the oracle DB).
	if tp.GasEstimation {
		return nil, nil
	}

	ev := evaluateRequire(ct)

	if !ev {
		msg := functionName + " condition not met"
		logger.Error(msg)
		return nil, vm.ErrExecutionReverted
	}

	return nil, nil
}

func Cast(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "cast"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	if !isValidType(input[32]) {
		logger.Error("invalid type to cast to")
		return nil, vm.ErrExecutionReverted
	}
	castToType := tfhe.UintType(input[32])

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, castToType)
	}

	ct := getCiphertext(state, tfhe.BytesToHash(input[0:32]), isTx)
	if ct == nil {
		logger.Error(functionName + " input not verified")
		return nil, vm.ErrExecutionReverted
	}

	res, err := ct.Cast(castToType)
	if err != nil {
		msg := functionName + " Run() error casting ciphertext to"
		logger.Error(msg, " type ", castToType)
		return nil, vm.ErrExecutionReverted
	}

	resHash := res.Hash()

	err = importCiphertext(state, res, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if shouldPrintPrecompileInfo(tp) {
		logger.Debug(functionName, " success",
			" ctHash ", resHash.Hex(),
		)
	}

	return resHash[:], nil
}

func TrivialEncrypt(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "trivialEncrypt"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	if len(input) != 33 {
		msg := functionName + " input len must be 33 bytes"
		logger.Error(msg, " input ", hex.EncodeToString(input), " len ", len(input))
		return nil, vm.ErrExecutionReverted
	}

	valueToEncrypt := *new(big.Int).SetBytes(input[0:32])
	encryptToType := tfhe.UintType(input[32])

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, encryptToType)
	}

	ct, err := tfhe.NewCipherTextTrivial(valueToEncrypt, encryptToType)
	if err != nil {
		logger.Error("failed to create trivial encrypted value")
		return nil, vm.ErrExecutionReverted
	}

	ctHash := ct.Hash()
	err = importCiphertext(state, ct, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if shouldPrintPrecompileInfo(tp) {
		logger.Debug(functionName, " success",
			" ctHash ", ctHash.Hex(),
			" valueToEncrypt ", valueToEncrypt.Uint64())
	}
	return ctHash[:], nil
}

func Div(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "div"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName, " inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Div(rhs)
	if err != nil {
		logger.Error(functionName, " failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName, " success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", ctHash.Hex())
	return ctHash[:], nil
}

func Gt(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: return ebool
	functionName := "gt"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName, " inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Gt(rhs)
	if err != nil {
		logger.Error(functionName, " failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName, " success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", ctHash.Hex())
	return ctHash[:], nil
}

func Gte(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: return ebool
	functionName := "gte"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Gte(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}
	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", ctHash.Hex())
	return ctHash[:], nil
}

func Rem(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "rem"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Rem(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}
	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", ctHash.Hex())
	return ctHash[:], nil
}

func And(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: bool math
	functionName := "and"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.And(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", ctHash.Hex())
	return ctHash[:], nil
}

func Or(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: bool math
	functionName := "or"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Or(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", ctHash.Hex())
	return ctHash[:], nil
}

func Xor(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: bool math
	functionName := "xor"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Xor(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}
	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", ctHash.Hex())
	return ctHash[:], nil
}

func Eq(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: bool math
	//solgen: return ebool
	functionName := "eq"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Eq(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}
	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), " result ", ctHash.Hex())
	return ctHash[:], nil
}

func Ne(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	//solgen: bool math
	//solgen: return ebool
	functionName := "ne"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Ne(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}
	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), "result", ctHash.Hex())
	return ctHash[:], nil
}

func Min(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "min"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Min(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}
	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), "result", ctHash.Hex())
	return ctHash[:], nil
}

func Max(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "max"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Max(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}
	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), "result", ctHash.Hex())
	return ctHash[:], nil
}

func Shl(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "shl"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Shl(rhs)
	if err != nil {
		logger.Error(functionName+" failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}
	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), "result", ctHash.Hex())
	return ctHash[:], nil
}

func Shr(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "shr"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("Starting new precompiled contract function ", functionName)
	}

	lhs, rhs, err := get2VerifiedOperands(state, input, isTx)
	if err != nil {
		logger.Error(functionName+" inputs not verified", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	if lhs.UintType != rhs.UintType {
		msg := functionName + " operand type mismatch"
		logger.Error(msg, " lhs ", lhs.UintType, " rhs ", rhs.UintType)
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, lhs.UintType)
	}

	result, err := lhs.Shr(rhs)
	if err != nil {
		logger.Error("fheShr failed", " err ", err)
		return nil, err
	}
	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	ctHash := result.Hash()

	logger.Debug(functionName+" success", " lhs ", lhs.Hash().Hex(), " rhs ", rhs.Hash().Hex(), "result", ctHash.Hex())
	return ctHash[:], nil
}

func Not(input []byte, tp *TxParams, state *FheosState) ([]byte, error) {
	functionName := "not"
	isTx := isTx(tp)

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new precompiled contract function ", functionName)
	}

	ct := getCiphertext(state, tfhe.BytesToHash(input[0:32]), isTx)
	if ct == nil {
		msg := "not unverified ciphertext handle"
		logger.Error(msg, "input", hex.EncodeToString(input))
		return nil, vm.ErrExecutionReverted
	}

	// If we are doing gas estimation, skip execution and insert a random ciphertext as a result.
	if tp.GasEstimation {
		return importRandomCiphertext(state, ct.UintType)
	}

	result, err := ct.Not()
	if err != nil {
		logger.Error("not failed", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	err = importCiphertext(state, result, isTx)
	if err != nil {
		logger.Error(functionName, " failed ", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	resultHash := result.Hash()
	logger.Debug(functionName+" success", " in ", ct.Hash().Hex(), " result ", resultHash.Hex())
	return resultHash[:], nil
}

func GetNetworkPublicKey(tp *TxParams, _ *FheosState) ([]byte, error) {
	functionName := "getNetworkPublicKey"

	if shouldPrintPrecompileInfo(tp) {
		logger.Info("starting new function get network public key:", functionName)
	}

	pk, err := tfhe.PublicKey()
	if err != nil {
		logger.Error("could not get public key", " err ", err)
		return nil, vm.ErrExecutionReverted
	}

	return pk, nil
}
