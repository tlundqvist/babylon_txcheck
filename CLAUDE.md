# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A lightweight CLI tool demonstrating Babylon-style Bitcoin staking transaction creation using only btcd libraries. This implementation includes a local copy of Babylon's `BuildStakingInfo` function and related code (extracted from the Babylon repository's `btcstaking` module) without the complex Cosmos SDK dependencies.

## Build and Run Commands

```bash
# Download dependencies
go mod download

# Build the binary
go build -o babylon_txcheck

# Run with required parameters (mainnet by default)
./babylon_txcheck \
  -staker-pk d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa \
  -fp-pk 4b15848e495a3a62283daaadb3f458a00859fe48e321f0121ebabbdd6698f9fa \
  -amount 1000000

# Run on testnet with custom staking time
./babylon_txcheck \
  -staker-pk d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa \
  -fp-pk 4b15848e495a3a62283daaadb3f458a00859fe48e321f0121ebabbdd6698f9fa \
  -amount 1000000 \
  -time 64000 \
  -testnet=true

# Required flags:
#   -staker-pk string   Staker public key (hex: 64 chars x-only or 66 chars compressed)
#   -fp-pk string       Finality provider public key (hex: 64 chars x-only or 66 chars compressed)
#   -amount int         Staking amount in satoshis
#
# Optional flags:
#   -time int           Staking time in blocks (default: uses API minimum of 64000 blocks)
#   -testnet bool       Use testnet parameters (default: false, uses mainnet)
#   -api string         Babylon API endpoint (default: "https://staking-api.babylonlabs.io/v2/network-info")
```

**Note:** The tool fetches parameters dynamically from the Babylon API on each run, including:
- Covenant public keys (9 keys with 6-of-9 quorum)
- Min/max staking amounts (500,000 to 500,000,000,000 satoshis)
- Min/max staking duration (64,000 blocks)
- Unbonding period (301 blocks)

Public keys can be provided in either format:
- **X-only format** (64 hex chars): `d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa`
- **Compressed format** (66 hex chars): `02d45c70d28f169e1f0c7f4a78e2bc73497afe585b70aa897955989068f3350aaa`

All user-provided values are validated against the API-fetched limits.

## Architecture

### Project Structure

The codebase consists of:
- `main.go`: CLI interface, parameter validation, and API integration
- `btcstaking/`: Local copy of Babylon's btcstaking module (extracted from Babylon repository)
  - `types.go`: Core `BuildStakingInfo` function and staking output construction
  - `scripts_utils.go`: Script building utilities (timeLock, unbonding, slashing paths)

This is a demonstration tool, not a production codebase. The `btcstaking` package is a minimal extraction from the Babylon repository to avoid Cosmos SDK dependencies.

### Core Bitcoin Staking Concept

The tool creates Taproot outputs with three distinct spending paths using Bitcoin script trees:

1. **TimeLock Path** (`buildTimeLockScript`): Staker-only spending after timelock expires
   - Script: `<Staker_PK> OP_CHECKSIGVERIFY <lockTime> OP_CHECKSEQUENCEVERIFY`
   - Uses CSV (CheckSequenceVerify) for relative timelocks

2. **Unbonding Path** (`buildUnbondingScript`): Early unbonding requiring staker + covenant signatures
   - Script: `<Staker_PK> OP_CHECKSIGVERIFY <Covenant_PK> OP_CHECKSIG`
   - Allows cooperative early exit

3. **Slashing Path** (`buildSlashingScript`): Requires staker + finality provider + covenant signatures
   - Script: `<Staker_PK> OP_CHECKSIGVERIFY <FP_PK> OP_CHECKSIGVERIFY <Covenant_PK> OP_CHECKSIG`
   - Enables slashing for protocol violations

### Taproot Implementation Pattern

The code demonstrates this critical pattern for Taproot script-path-only outputs:

1. Build individual leaf scripts for each spending condition
2. Create `TapLeaf` structures with `txscript.NewBaseTapLeaf()`
3. Assemble into Merkle tree with `txscript.AssembleTaprootScriptTree()`
4. Use standard unspendable internal key (`0250929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac0`)
5. Compute output key with `txscript.ComputeTaprootOutputKey(internalPubKey, tapScriptRootHash)`
6. Each spending path requires the script + control block (derived from Merkle proof)

### Key Technical Details

- **X-only public keys**: All Taproot scripts use x-only pubkeys (32 bytes), extracted via `SerializeCompressed()[1:]`
- **Control blocks**: Generated from Merkle proofs using `proof.ToControlBlock(internalPubKey)` for each spending path
- **Network parameters**: Uses `chaincfg.TestNet3Params` or `chaincfg.MainNetParams` depending on `-testnet` flag

## Dependencies

- `btcsuite/btcd` v0.25.0: Core Bitcoin operations
- `btcsuite/btcd/btcec/v2` v2.3.6: Elliptic curve cryptography
- `btcsuite/btcd/btcutil` v1.1.6: Bitcoin utilities
- `btcsuite/btcd/chaincfg/chainhash` v1.1.0: Chain configuration and hashing

All dependencies are standard btcd libraries - no Babylon/Cosmos SDK complexity.

**Module name:** `babylon_txcheck` (as defined in go.mod)

## Important Notes for Development

- The tool fetches real covenant public keys from the Babylon API on every run
- Staker and finality provider public keys must be provided via CLI flags (NOT generated)
- The tool creates staking outputs only - no actual transactions are broadcast
- Always test on testnet before mainnet operations
- All parameters are validated against live Babylon network limits fetched from the API
- Public keys are accepted in both x-only (64 chars) and compressed (66 chars) hex formats
- Covenant quorum is determined by the Babylon API, not configurable via CLI
- This code is for educational/testing purposes; refer to the Babylon repository for production implementations
