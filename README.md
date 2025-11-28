# Babylon BTC Staking Transaction Builder

A CLI tool demonstrating Babylon-style Bitcoin staking transaction creation using only btcd libraries. This implementation includes a local copy of Babylon's `BuildStakingInfo` function (extracted from the Babylon repository) without the complex Cosmos SDK dependencies.

## Overview

This tool demonstrates how to create Bitcoin staking outputs with Taproot scripts compatible with the Babylon staking protocol. The staking output includes three distinct spending paths:

1. **TimeLock Path**: Staker can spend after the staking time expires (normal unbonding)
2. **Unbonding Path**: Staker + Covenant committee can spend anytime (early unbonding)
3. **Slashing Path**: Staker + Finality Provider + Covenant can spend anytime (for slashing)

## Quick Start

### Prerequisites

- Go 1.23 or later
- Internet connection (to fetch Babylon network parameters from API)

### Installation

```bash
cd ~/src/babylon_txcheck
go mod download
go build -o babylon_txcheck
```

### Basic Usage

The tool requires three parameters: staker public key, finality provider public key, and staking amount.

```bash
./babylon_txcheck \
  -staker-pk d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa \
  -fp-pk 4b15848e495a3a62283daaadb3f458a00859fe48e321f0121ebabbdd6698f9fa \
  -amount 1000000
```

This creates a mainnet staking output for 1,000,000 satoshis (0.01 BTC) using the API-defined minimum staking time (64,000 blocks).

### Finding Finality Provider Keys

Use `fetch_fp.py` to interactively select a finality provider and get their BTC public key:

```bash
pip install requests wcwidth
python3 fetch_fp.py
```

- Enter a **number** to select a provider, or **text** to search/filter by moniker
- Filtered results keep their original numbers for easy reference
- Selected provider's BTC public key is displayed for use with `-fp-pk` flag

### Testnet Usage

```bash
./babylon_txcheck \
  -staker-pk d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa \
  -fp-pk 4b15848e495a3a62283daaadb3f458a00859fe48e321f0121ebabbdd6698f9fa \
  -amount 1000000 \
  -time 64000 \
  -testnet=true
```

## Command Line Options

### Required Flags

- `-staker-pk <hex>`: Staker's public key (64-char x-only or 66-char compressed format)
- `-fp-pk <hex>`: Finality provider's public key (64-char x-only or 66-char compressed format)
- `-amount <satoshis>`: Staking amount in satoshis

### Optional Flags

- `-time <blocks>`: Staking duration in blocks (default: uses API minimum of 64,000 blocks)
- `-testnet`: Use Bitcoin testnet parameters (default: false, uses mainnet)
- `-api <url>`: Babylon API endpoint (default: `https://babylon.nodes.guru/babylon/btcstaking/v1/params`)

### Public Key Formats

Public keys can be provided in either format:

- **X-only** (64 hex characters): `d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa`
- **Compressed** (66 hex characters): `02d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa`

## Output

The tool displays:

1. **Network Configuration**: Mainnet or testnet
2. **Staking Parameters**: Amount, duration (in blocks, days, weeks, months)
3. **Keys Summary**: Staker, finality provider, and all covenant committee keys
4. **Spending Paths**: Scripts and control blocks for each of the three spending conditions
5. **Staking Output**: Taproot address, PkScript hex, and output value

Example output:

```
=== Babylon-Style BTC Staking Transaction Builder ===
Fetching parameters from Babylon API: https://babylon.nodes.guru/babylon/btcstaking/v1/params
✓ Successfully fetched parameters (Covenant quorum: 6/9)

Staking Amount: 1000000 satoshis
Staking Time: 64000 blocks (≈ 444.4 days / 63.5 weeks / 14.6 months)
Network: Mainnet

Keys Summary:
  Staker PK: d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa
  Finality Provider PK: 4b15848e495a3a62283daaadb3f458a00859fe48e321f0121ebabbdd6698f9fa
  Covenant Committee: 9 keys (quorum: 6)
    [1] 03f3f34e9d5f8e5c...
    [2] 024a7c8b9e3f2d1a...
    ...

✓ Successfully built staking output using Babylon's BuildStakingInfo!

Spending Paths:
  1. TimeLock Path (normal unbonding after staking time):
     Script: 20d45c70...ac00a09f
     Control Block: c150929b...

  2. Unbonding Path (early unbonding with covenant cooperation):
     Script: 20d45c70...21ac
     Control Block: c150929b...

  3. Slashing Path (slashing with FP and covenant cooperation):
     Script: 20d45c70...ac21ac
     Control Block: c150929b...

Staking Output Information:
  Value: 1000000 satoshis
  Taproot Address: bc1p...
  PkScript (hex): 5120a7f8c3...
  PkScript Length: 34 bytes

=== Success! ===
Taproot staking output created successfully.
```

## Architecture

### Project Structure

```
babylon_txcheck/
├── main.go              # CLI interface, API integration, parameter validation
├── btcstaking/          # Local copy of Babylon's btcstaking module
│   ├── types.go         # Core BuildStakingInfo function
│   └── scripts_utils.go # Script building utilities
├── go.mod               # Go module dependencies
├── go.sum               # Dependency checksums
├── CLAUDE.md            # Development guidance for Claude Code
└── README.md            # This file
```

### How It Works

1. **Fetch Parameters**: Retrieves live network parameters from Babylon API:
   - 9 covenant public keys with 6-of-9 quorum requirement
   - Min/max staking amounts (500,000 to 500,000,000,000 satoshis)
   - Min/max staking duration (64,000 blocks)

2. **Validate Inputs**: Ensures user-provided values meet network requirements

3. **Build Scripts**: Creates three Bitcoin scripts for different spending conditions:
   - `buildTimeLockScript()`: CSV timelock for normal unbonding
   - `buildUnbondingScript()`: Multi-sig for early exit with covenant
   - `buildSlashingScript()`: Multi-sig for slashing events

4. **Assemble Taproot Tree**: Combines scripts into Merkle tree with standard unspendable internal key

5. **Generate Output**: Creates Taproot address and outputs spending path information

### Taproot Implementation

The tool demonstrates key Taproot concepts:

- **Script-path-only outputs**: Uses standard unspendable internal key (`0250929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac0`)
- **X-only public keys**: Converts compressed pubkeys to x-only format (32 bytes) for Taproot scripts
- **Control blocks**: Generated from Merkle proofs for each spending path
- **Relative timelocks**: Uses CSV (CheckSequenceVerify) for time-bound spending conditions

## API Integration

The tool fetches parameters from the Babylon API on every run to ensure compliance with current network rules:

- **Default API**: `https://babylon.nodes.guru/babylon/btcstaking/v1/params`
- **Custom API**: Use `-api` flag to specify alternative endpoint
- **Parameters fetched**: Covenant keys, quorum, amount limits, time limits

All user inputs are validated against these fetched limits before creating the staking output.

## Parameter Limits

Current Babylon mainnet limits (as of API fetch):

- **Min staking amount**: 500,000 satoshis (0.005 BTC)
- **Max staking amount**: 500,000,000,000 satoshis (5,000 BTC)
- **Staking duration**: 64,000 blocks (≈ 444 days)
- **Covenant quorum**: 6 of 9 signatures required

*Note: These values are fetched dynamically and may change.*

## Development Notes

### Local btcstaking Package

The `btcstaking/` directory contains a minimal extraction from the [Babylon repository](https://github.com/babylonlabs-io/babylon) to avoid pulling in the entire Cosmos SDK dependency tree. This includes:

- Core staking output construction logic
- Script building utilities
- Taproot tree assembly

### Dependencies

All Bitcoin operations use standard btcd libraries:

- `btcsuite/btcd` v0.25.0
- `btcsuite/btcd/btcec/v2` v2.3.6
- `btcsuite/btcd/btcutil` v1.1.6
- `btcsuite/btcd/chaincfg/chainhash` v1.1.0

No Cosmos SDK or Babylon-specific dependencies required.

### Limitations

- **Output only**: Tool creates staking outputs but does NOT create or broadcast complete transactions
- **No key generation**: Public keys must be provided externally
- **No signing**: Tool does not handle private keys or transaction signing
- **Educational purpose**: Not intended for production use

## Testing

### Generate Test Keys

Use `btcd` tools or any Bitcoin key generator:

```bash
# Example x-only pubkey (not a real key - do not use)
d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa
```

### Test on Testnet First

Always use `-testnet=true` when testing:

```bash
./babylon_txcheck \
  -staker-pk <your-test-key> \
  -fp-pk <test-fp-key> \
  -amount 500000 \
  -testnet=true
```

## References

- [Babylon Protocol Documentation](https://docs.babylonlabs.io/)
- [Babylon GitHub Repository](https://github.com/babylonlabs-io/babylon)
- [Bitcoin Taproot (BIP 341)](https://github.com/bitcoin/bips/blob/master/bip-0341.mediawiki)
- [Babylon Staking Script Documentation](https://github.com/babylonlabs-io/babylon/blob/main/docs/staking-script.md)

## License

This is a demonstration tool for educational purposes. Refer to the Babylon repository for licensing information on the underlying protocol.

## Security Warnings

- Never use this tool with real private keys
- Always test on testnet before mainnet
- Verify all outputs independently before broadcasting any transactions
- This tool is for educational purposes only - use at your own risk
