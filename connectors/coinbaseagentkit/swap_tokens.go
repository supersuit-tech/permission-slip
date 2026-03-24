package coinbaseagentkit

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/coinbase/cdp-sdk/go/openapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type swapTokensParams struct {
	Network        string  `json:"network"`
	FromToken      string  `json:"from_token"`
	ToToken        string  `json:"to_token"`
	FromAmount     string  `json:"from_amount"`
	Taker          string  `json:"taker"`
	SlippageBps    *int    `json:"slippage_bps"`
	GasPrice       *string `json:"gas_price"`
	IdempotencyKey *string `json:"idempotency_key"`
}

type swapTokensAction struct {
	conn *CoinbaseAgentKitConnector
}

func (a *swapTokensAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var p swapTokensParams
	if err := json.Unmarshal(req.Parameters, &p); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	net, err := parseSwapNetwork(p.Network)
	if err != nil {
		return nil, err
	}
	fromTok := strings.TrimSpace(p.FromToken)
	toTok := strings.TrimSpace(p.ToToken)
	if !evmAddrRE.MatchString(fromTok) || !evmAddrRE.MatchString(toTok) {
		return nil, &connectors.ValidationError{Message: "from_token and to_token must be 0x-prefixed 40-hex-character contract addresses (use 0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE for native per EIP-7528)"}
	}
	fromAmt := strings.TrimSpace(p.FromAmount)
	if fromAmt == "" {
		return nil, &connectors.ValidationError{Message: "from_amount is required (atomic units as a decimal string)"}
	}
	taker := strings.TrimSpace(p.Taker)
	if !evmAddrRE.MatchString(taker) {
		return nil, &connectors.ValidationError{Message: "taker must be the 0x-prefixed wallet address that will execute the swap"}
	}

	body := openapi.CreateEvmSwapQuoteJSONRequestBody{
		Network:    net,
		FromAmount: fromAmt,
		FromToken:  fromTok,
		ToToken:    toTok,
		Taker:      taker,
	}
	if p.SlippageBps != nil {
		body.SlippageBps = p.SlippageBps
	}
	if p.GasPrice != nil && strings.TrimSpace(*p.GasPrice) != "" {
		gp := strings.TrimSpace(*p.GasPrice)
		body.GasPrice = &gp
	}

	var quoteParams *openapi.CreateEvmSwapQuoteParams
	if p.IdempotencyKey != nil && strings.TrimSpace(*p.IdempotencyKey) != "" {
		ik := openapi.IdempotencyKey(strings.TrimSpace(*p.IdempotencyKey))
		quoteParams = &openapi.CreateEvmSwapQuoteParams{XIdempotencyKey: &ik}
	}

	cdpClient, err := newCDPClient(req.Credentials)
	if err != nil {
		return nil, err
	}

	qResp, err := cdpClient.CreateEvmSwapQuoteWithResponse(ctx, quoteParams, body)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("CDP swap quote timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("CDP swap quote failed: %v", err)}
	}
	if qResp.StatusCode() != 201 || qResp.JSON201 == nil {
		return nil, mapCDPError(qResp.StatusCode(), qResp.Body)
	}

	quote, err := qResp.JSON201.AsCreateSwapQuoteResponse()
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("parse swap quote response: %v", err)}
	}
	if !bool(quote.LiquidityAvailable) {
		return nil, &connectors.ValidationError{Message: "swap quote reports no liquidity available for this pair and amount"}
	}

	chainID, err := chainIDForSwapNetwork(net)
	if err != nil {
		return nil, err
	}
	rpcURL, ok := publicRPCForSwapNetwork(net)
	if !ok {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("no public RPC configured for swap network %s", net)}
	}
	ethCl, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("dial RPC: %v", err)}
	}
	defer ethCl.Close()

	takerAddr := common.HexToAddress(taker)
	nonce, err := ethCl.PendingNonceAt(ctx, takerAddr)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("fetch pending nonce: %v", err)}
	}

	tip, gasFeeCap, err := suggestEIP1559Fees(ctx, ethCl)
	if err != nil {
		return nil, &connectors.ExternalError{Message: err.Error()}
	}
	tip, gasFeeCap = adjustFeeCapsFromQuoteGasPrice(tip, gasFeeCap, quote.Transaction.GasPrice)

	gasLimit, err := parseUint64String(quote.Transaction.Gas, "gas limit")
	if err != nil {
		return nil, err
	}
	toContract := common.HexToAddress(quote.Transaction.To)
	value, err := parseBigIntStringRequired(quote.Transaction.Value, "transaction value")
	if err != nil {
		return nil, err
	}
	data := common.FromHex(quote.Transaction.Data)
	if err := validateSwapQuoteTx(quote.Transaction.To, gasLimit, data); err != nil {
		return nil, err
	}

	unsigned := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &toContract,
		Value:     value,
		Data:      data,
	})

	rlpRaw, err := unsigned.MarshalBinary()
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("serialize unsigned swap tx: %v", err)}
	}
	rlpHex := "0x" + common.Bytes2Hex(rlpRaw)

	signResp, err := cdpClient.SignEvmTransactionWithResponse(ctx, taker, nil, openapi.SignEvmTransactionJSONRequestBody{
		Transaction: rlpHex,
	})
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("CDP sign timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("CDP sign failed: %v", err)}
	}
	if signResp.StatusCode() != 200 || signResp.JSON200 == nil {
		return nil, mapCDPError(signResp.StatusCode(), signResp.Body)
	}
	signedHex := signResp.JSON200.SignedTransaction
	if !strings.HasPrefix(signedHex, "0x") {
		signedHex = "0x" + signedHex
	}

	sendNet, err := swapToSendNetwork(net)
	if err != nil {
		return nil, err
	}

	sendResp, err := cdpClient.SendEvmTransactionWithResponse(ctx, taker, nil, openapi.SendEvmTransactionJSONRequestBody{
		Network:     sendNet,
		Transaction: signedHex,
	})
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("CDP send timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("CDP send failed: %v", err)}
	}
	if sendResp.StatusCode() != 200 || sendResp.JSON200 == nil {
		return nil, mapCDPError(sendResp.StatusCode(), sendResp.Body)
	}

	return jsonResult(map[string]any{
		"transaction_hash": sendResp.JSON200.TransactionHash,
		"network":          string(net),
		"taker":            taker,
		"from_amount":      quote.FromAmount,
		"to_amount":        quote.ToAmount,
		"min_to_amount":    quote.MinToAmount,
		"from_token":       quote.FromToken,
		"to_token":         quote.ToToken,
		"block_number":     quote.BlockNumber,
	})
}

func chainIDForSwapNetwork(n openapi.EvmSwapsNetwork) (*big.Int, error) {
	switch n {
	case openapi.EvmSwapsNetworkBase:
		return big.NewInt(8453), nil
	case openapi.EvmSwapsNetworkEthereum:
		return big.NewInt(1), nil
	case openapi.EvmSwapsNetworkPolygon:
		return big.NewInt(137), nil
	case openapi.EvmSwapsNetworkArbitrum:
		return big.NewInt(42161), nil
	case openapi.EvmSwapsNetworkOptimism:
		return big.NewInt(10), nil
	default:
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("unsupported swap network: %s", n)}
	}
}

func swapToSendNetwork(n openapi.EvmSwapsNetwork) (openapi.SendEvmTransactionJSONBodyNetwork, error) {
	switch n {
	case openapi.EvmSwapsNetworkArbitrum:
		return openapi.SendEvmTransactionJSONBodyNetworkArbitrum, nil
	case openapi.EvmSwapsNetworkBase:
		return openapi.SendEvmTransactionJSONBodyNetworkBase, nil
	case openapi.EvmSwapsNetworkEthereum:
		return openapi.SendEvmTransactionJSONBodyNetworkEthereum, nil
	case openapi.EvmSwapsNetworkOptimism:
		return openapi.SendEvmTransactionJSONBodyNetworkOptimism, nil
	case openapi.EvmSwapsNetworkPolygon:
		return openapi.SendEvmTransactionJSONBodyNetworkPolygon, nil
	default:
		return "", &connectors.ValidationError{Message: fmt.Sprintf("cannot map swap network to send network: %s", n)}
	}
}

func parseUint64String(s, field string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, &connectors.ValidationError{Message: fmt.Sprintf("missing %s", field)}
	}
	v := new(big.Int)
	if _, ok := v.SetString(s, 10); !ok || v.Sign() <= 0 {
		return 0, &connectors.ValidationError{Message: fmt.Sprintf("invalid %s", field)}
	}
	if !v.IsUint64() {
		return 0, &connectors.ValidationError{Message: fmt.Sprintf("%s overflows uint64", field)}
	}
	return v.Uint64(), nil
}

func parseBigIntString(s, field string) (*big.Int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	v := new(big.Int)
	if _, ok := v.SetString(s, 10); !ok {
		return nil, false
	}
	return v, true
}

func parseBigIntStringRequired(s, field string) (*big.Int, error) {
	v, ok := parseBigIntString(s, field)
	if !ok {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid or missing %s", field)}
	}
	return v, nil
}
