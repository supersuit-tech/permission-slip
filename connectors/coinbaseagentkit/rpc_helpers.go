package coinbaseagentkit

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/coinbase/cdp-sdk/go/openapi"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// Minimal ERC-20 transfer ABI for encoding calldata (send_crypto).
const erc20TransferABI = `[{"name":"transfer","type":"function","inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}]`

// Limits for swap quote → unsigned tx construction (defense in depth; CDP also validates).
const (
	maxSwapCalldataBytes = 256 * 1024
	maxSwapGasLimit      = 16_000_000
)

// suggestEIP1559Fees returns tip (priority fee) and gasFeeCap for a DynamicFeeTx using only
// public RPC reads. When the chain exposes EIP-1559 base fee, gasFeeCap is 2*baseFee+tip;
// otherwise falls back to SuggestGasPrice.
func suggestEIP1559Fees(ctx context.Context, ethCl *ethclient.Client) (tip, gasFeeCap *big.Int, err error) {
	tip, err = ethCl.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("suggest gas tip: %w", err)
	}
	header, err := ethCl.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch chain head: %w", err)
	}
	if header.BaseFee != nil {
		gasFeeCap = new(big.Int).Add(new(big.Int).Mul(header.BaseFee, big.NewInt(2)), tip)
	} else {
		gasFeeCap, err = ethCl.SuggestGasPrice(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("suggest gas price: %w", err)
		}
	}
	return tip, gasFeeCap, nil
}

// adjustFeeCapsFromQuoteGasPrice raises gasFeeCap (and tip if needed) when the CDP swap quote
// specifies a higher minimum max fee per gas.
func adjustFeeCapsFromQuoteGasPrice(tip, gasFeeCap *big.Int, quoteGasPrice string) (*big.Int, *big.Int) {
	qGas, ok := parseBigIntString(quoteGasPrice, "quote gas price")
	if !ok || qGas.Cmp(gasFeeCap) <= 0 {
		return tip, gasFeeCap
	}
	gasFeeCap = new(big.Int).Set(qGas)
	if gasFeeCap.Cmp(tip) < 0 {
		tip = new(big.Int).Set(gasFeeCap)
	}
	return tip, gasFeeCap
}

func validateSwapQuoteTx(toAddr string, gasLimit uint64, data []byte) error {
	if strings.TrimSpace(toAddr) == "" {
		return &connectors.ValidationError{Message: "swap quote transaction has empty contract address"}
	}
	if !evmAddrRE.MatchString(strings.TrimSpace(toAddr)) {
		return &connectors.ValidationError{Message: "swap quote transaction \"to\" is not a valid 0x-prefixed address"}
	}
	if gasLimit == 0 || gasLimit > maxSwapGasLimit {
		return &connectors.ValidationError{Message: fmt.Sprintf("swap quote gas limit %d is invalid or exceeds maximum %d", gasLimit, maxSwapGasLimit)}
	}
	if len(data) > maxSwapCalldataBytes {
		return &connectors.ValidationError{Message: fmt.Sprintf("swap quote calldata exceeds maximum size (%d bytes)", maxSwapCalldataBytes)}
	}
	return nil
}

var (
	erc20ABIOnce   sync.Once
	erc20ABIParsed abi.ABI
	erc20ABIErr    error
)

// parsedErc20TransferABI parses the minimal ERC-20 transfer ABI once per process.
func parsedErc20TransferABI() (*abi.ABI, error) {
	erc20ABIOnce.Do(func() {
		erc20ABIParsed, erc20ABIErr = abi.JSON(strings.NewReader(erc20TransferABI))
	})
	if erc20ABIErr != nil {
		return nil, erc20ABIErr
	}
	return &erc20ABIParsed, nil
}

func publicRPCForSendNetwork(n openapi.SendEvmTransactionJSONBodyNetwork) (string, bool) {
	switch n {
	case openapi.SendEvmTransactionJSONBodyNetworkBase:
		return "https://mainnet.base.org", true
	case openapi.SendEvmTransactionJSONBodyNetworkBaseSepolia:
		return "https://sepolia.base.org", true
	case openapi.SendEvmTransactionJSONBodyNetworkEthereum:
		return "https://ethereum.publicnode.com", true
	case openapi.SendEvmTransactionJSONBodyNetworkEthereumSepolia:
		return "https://ethereum-sepolia.publicnode.com", true
	case openapi.SendEvmTransactionJSONBodyNetworkPolygon:
		return "https://polygon-rpc.com", true
	case openapi.SendEvmTransactionJSONBodyNetworkArbitrum:
		return "https://arb1.arbitrum.io/rpc", true
	case openapi.SendEvmTransactionJSONBodyNetworkOptimism:
		return "https://mainnet.optimism.io", true
	case openapi.SendEvmTransactionJSONBodyNetworkAvalanche:
		return "https://api.avax.network/ext/bc/C/rpc", true
	default:
		return "", false
	}
}

func publicRPCForSwapNetwork(n openapi.EvmSwapsNetwork) (string, bool) {
	switch n {
	case openapi.EvmSwapsNetworkBase:
		return "https://mainnet.base.org", true
	case openapi.EvmSwapsNetworkEthereum:
		return "https://ethereum.publicnode.com", true
	case openapi.EvmSwapsNetworkPolygon:
		return "https://polygon-rpc.com", true
	case openapi.EvmSwapsNetworkArbitrum:
		return "https://arb1.arbitrum.io/rpc", true
	case openapi.EvmSwapsNetworkOptimism:
		return "https://mainnet.optimism.io", true
	default:
		return "", false
	}
}
