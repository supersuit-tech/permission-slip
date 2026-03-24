package coinbaseagentkit

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/coinbase/cdp-sdk/go/openapi"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type sendCryptoParams struct {
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	Network     string `json:"network"`
	// AmountWei is the amount in the smallest unit (wei for native ETH, base units for ERC-20).
	AmountWei string `json:"amount_wei"`
	// TokenContract is optional. When empty, sends native gas token (ETH / equivalent).
	TokenContract *string `json:"token_contract"`
}

type sendCryptoAction struct {
	conn *CoinbaseAgentKitConnector
}

func (a *sendCryptoAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var p sendCryptoParams
	if err := json.Unmarshal(req.Parameters, &p); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if !evmAddrRE.MatchString(strings.TrimSpace(p.FromAddress)) {
		return nil, &connectors.ValidationError{Message: "from_address must be a 0x-prefixed 40-hex-character EVM address"}
	}
	if !evmAddrRE.MatchString(strings.TrimSpace(p.ToAddress)) {
		return nil, &connectors.ValidationError{Message: "to_address must be a 0x-prefixed 40-hex-character EVM address"}
	}
	amountWei := strings.TrimSpace(p.AmountWei)
	if amountWei == "" {
		return nil, &connectors.ValidationError{Message: "amount_wei is required"}
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(amountWei, 10); !ok || amt.Sign() <= 0 {
		return nil, &connectors.ValidationError{Message: "amount_wei must be a positive decimal integer string"}
	}

	net, err := parseSendNetwork(p.Network)
	if err != nil {
		return nil, err
	}

	fromAddr := common.HexToAddress(strings.TrimSpace(p.FromAddress))
	toAddr := common.HexToAddress(strings.TrimSpace(p.ToAddress))

	rpcURL, ok := publicRPCForSendNetwork(net)
	if !ok {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("no public RPC configured for network %s", net)}
	}

	ethCl, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("dial RPC for fee estimation: %v", err)}
	}
	defer ethCl.Close()

	chainID, err := chainIDForSendNetwork(net)
	if err != nil {
		return nil, err
	}

	nonce, err := ethCl.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("fetch pending nonce: %v", err)}
	}

	tip, gasFeeCap, err := suggestEIP1559Fees(ctx, ethCl)
	if err != nil {
		return nil, &connectors.ExternalError{Message: err.Error()}
	}

	var tx *types.Transaction
	var gasLimit uint64

	if p.TokenContract == nil || strings.TrimSpace(*p.TokenContract) == "" {
		gasLimit = 21000
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			GasTipCap: tip,
			GasFeeCap: gasFeeCap,
			Gas:       gasLimit,
			To:        &toAddr,
			Value:     amt,
			Data:      nil,
		})
	} else {
		tokenStr := strings.TrimSpace(*p.TokenContract)
		if !evmAddrRE.MatchString(tokenStr) {
			return nil, &connectors.ValidationError{Message: "token_contract must be a 0x-prefixed 40-hex-character contract address"}
		}
		tokenAddr := common.HexToAddress(tokenStr)
		parsed, err := parsedErc20TransferABI()
		if err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("parse ERC-20 ABI: %v", err)}
		}
		data, err := parsed.Pack("transfer", toAddr, amt)
		if err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("encode transfer calldata: %v", err)}
		}
		gasLimit, err = ethCl.EstimateGas(ctx, ethereum.CallMsg{
			From: fromAddr,
			To:   &tokenAddr,
			Data: data,
		})
		if err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("estimate gas for token transfer: %v", err)}
		}
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			GasTipCap: tip,
			GasFeeCap: gasFeeCap,
			Gas:       gasLimit,
			To:        &tokenAddr,
			Value:     big.NewInt(0),
			Data:      data,
		})
	}

	rlpRaw, err := tx.MarshalBinary()
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("serialize unsigned transaction: %v", err)}
	}
	rlpHex := "0x" + common.Bytes2Hex(rlpRaw)

	cdpClient, err := newCDPClient(req.Credentials)
	if err != nil {
		return nil, err
	}

	signResp, err := cdpClient.SignEvmTransactionWithResponse(
		ctx,
		strings.TrimSpace(p.FromAddress),
		nil,
		openapi.SignEvmTransactionJSONRequestBody{Transaction: rlpHex},
	)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("CDP sign timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("CDP sign request failed: %v", err)}
	}
	if signResp.StatusCode() != 200 || signResp.JSON200 == nil {
		return nil, mapCDPError(signResp.StatusCode(), signResp.Body)
	}

	signedHex := signResp.JSON200.SignedTransaction
	if !strings.HasPrefix(signedHex, "0x") {
		signedHex = "0x" + signedHex
	}
	sendResp, err := cdpClient.SendEvmTransactionWithResponse(
		ctx,
		strings.TrimSpace(p.FromAddress),
		nil,
		openapi.SendEvmTransactionJSONRequestBody{
			Network:     net,
			Transaction: signedHex,
		},
	)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("CDP send timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("CDP send request failed: %v", err)}
	}
	if sendResp.StatusCode() != 200 || sendResp.JSON200 == nil {
		return nil, mapCDPError(sendResp.StatusCode(), sendResp.Body)
	}

	return jsonResult(map[string]any{
		"transaction_hash": sendResp.JSON200.TransactionHash,
		"from_address":     strings.TrimSpace(p.FromAddress),
		"to_address":       strings.TrimSpace(p.ToAddress),
		"network":          string(net),
	})
}
