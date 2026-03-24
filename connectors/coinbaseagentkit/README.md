# Coinbase AgentKit (CDP Server Wallet) connector

Integrates [Coinbase Developer Platform](https://docs.cdp.coinbase.com/) Wallet API v2 via the official [`github.com/coinbase/cdp-sdk/go`](https://github.com/coinbase/cdp-sdk) client (JWT API key auth + wallet secret for signing).

## Stored credentials

| Key | Description |
|-----|-------------|
| `api_key_id` | CDP API Key ID from the portal |
| `api_key_secret` | CDP API Key Secret (PEM EC key or Ed25519 base64, per portal download) |
| `wallet_secret` | Wallet secret from the Wallet API product (base64 PKCS#8) |
| `base_path` | Optional; override API base URL (tests only) |

Create keys at [portal.cdp.coinbase.com](https://portal.cdp.coinbase.com/).

## Actions

- `coinbase_agentkit.create_evm_account` — create MPC-managed EVM account
- `coinbase_agentkit.list_evm_accounts` — paginated account list
- `coinbase_agentkit.send_crypto` — native or ERC-20 transfer (requires payment method)
- `coinbase_agentkit.swap_tokens` — DEX swap via quote + sign + send (requires payment method)
- `coinbase_agentkit.request_testnet_funds` — faucet
- `coinbase_agentkit.list_token_balances` — balances on Base / Base Sepolia / Ethereum
- `coinbase_agentkit.get_swap_price` — read-only swap quote

`send_crypto` and `swap_tokens` use public RPC endpoints only for nonce and fee estimation; signing and broadcast go through CDP.

Before signing a swap from a CDP quote, the connector validates the quoted transaction’s `to` address format, gas limit (must be positive and at most 16M), and calldata length (at most 256 KiB) so obviously malformed quotes fail fast.
