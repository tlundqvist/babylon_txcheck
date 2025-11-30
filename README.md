# Babylon BTC Staking Transaction Builder

A CLI tool demonstrating Babylon-style Bitcoin staking transaction
creation. Useful for educational purposes or as a tool to double check
the staking taproot address when staking BTC on the Babylon network.

This tool uses only the btcd libraries. It also includes a local copy
of Babylon's `BuildStakingInfo` function (extracted from the Babylon
repository) without the complex Cosmos SDK dependencies.

## Overview

This tool demonstrates how to create the Bitcoin staking output with
Taproot scripts compatible with the Babylon staking protocol. The
staking output includes three distinct spending paths:

1. **TimeLock Path**: Staker can spend after the staking time expires (normal unbonding)
2. **Unbonding Path**: Staker + Covenant committee can spend anytime (early unbonding)
3. **Slashing Path**: Staker + Finality Provider + Covenant can spend anytime (for slashing)

The tool calculates and outputs:
- The taproot staking script and the resulting taproot address of the staking output of the transaction.
- Not the full transaction. A full transaction would also include spending inputs and maybe a change output.
- Only staking tx. The two presigned slashing transactions used in BTC staking for Babylon are not included.

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

The tool requires three parameters: staker public key, finality
provider public key, and staking amount.

```bash
./babylon_txcheck \
  -staker-pk d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa \
  -fp-pk 4b15848e495a3a62283daaadb3f458a00859fe48e321f0121ebabbdd6698f9fa \
  -amount 1000000
```

This creates a mainnet staking output for 1,000,000 satoshis (0.01
BTC) using the API-defined minimum staking time (64,000 blocks).

### Finding your own staking public key

Your public key can maybe be found in your wallet software. Otherwise
it is revealed in a spend transaction, so look at an input in an old
transaction where your address is used.

### Finding Finality Provider Keys

Use `fetch_fp.py` to interactively select a finality provider and get
their BTC public key:

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
- `-api <url>`: Babylon API endpoint (default: `https://staking-api.babylonlabs.io/v2/network-info`)

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
Fetching parameters from Babylon API: https://staking-api.babylonlabs.io/v2/network-info
✓ Successfully fetched parameters (Version: 2, Covenant quorum: 6/9)

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
   - Covenant public keys with quorum requirement
   - Min/max staking amounts
   - Min/max staking duration

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

## References

- [Babylon Protocol Documentation](https://docs.babylonlabs.io/)
- [Babylon GitHub Repository](https://github.com/babylonlabs-io/babylon)
- [Bitcoin Taproot (BIP 341)](https://github.com/bitcoin/bips/blob/master/bip-0341.mediawiki)
- [Babylon Staking Script Documentation](https://github.com/babylonlabs-io/babylon/blob/main/docs/staking-script.md)

## License & Attribution

This project uses a dual-license structure:

- **Main tool code** (`main.go`, `fetch_fp.py`, etc.): MIT License - see root `LICENSE` file
- **btcstaking/ directory**: Contains code extracted from the [Babylon repository](https://github.com/babylonlabs-io/babylon), licensed under Business Source License 1.1 - see `btcstaking/LICENSE`

This tool is for educational/demonstration purposes and integrates with the Babylon Protocol.

## Security Warnings

- Verify all outputs independently before broadcasting any transactions
- Use at your own risk!
