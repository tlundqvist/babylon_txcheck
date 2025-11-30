package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"babylon_txcheck/btcstaking"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

// BabylonVersionedParams holds a single version of staking parameters
type BabylonVersionedParams struct {
	Version              int      `json:"version"`
	CovenantPks          []string `json:"covenant_pks"`
	CovenantQuorum       uint32   `json:"covenant_quorum"`
	MinStakingValueSat   int64    `json:"min_staking_value_sat"`
	MaxStakingValueSat   int64    `json:"max_staking_value_sat"`
	MinStakingTimeBlocks uint32   `json:"min_staking_time_blocks"`
	MaxStakingTimeBlocks uint32   `json:"max_staking_time_blocks"`
	UnbondingTimeBlocks  uint32   `json:"unbonding_time_blocks"`
}

// BabylonParams holds the parameters fetched from the Babylon API
type BabylonParams struct {
	Data struct {
		Params struct {
			Bbn []BabylonVersionedParams `json:"bbn"`
		} `json:"params"`
	} `json:"data"`
}

// GetLatestParams returns the highest version parameters
func (bp *BabylonParams) GetLatestParams() *BabylonVersionedParams {
	if len(bp.Data.Params.Bbn) == 0 {
		return nil
	}

	// Find the version with the highest version number
	latest := &bp.Data.Params.Bbn[0]
	for i := range bp.Data.Params.Bbn {
		if bp.Data.Params.Bbn[i].Version > latest.Version {
			latest = &bp.Data.Params.Bbn[i]
		}
	}
	return latest
}

// parsePubKey parses a public key from hex string, supporting both:
// - Compressed format (66 hex chars): 02/03 prefix + 32 bytes
// - X-only format (64 hex chars): 32 bytes only
func parsePubKey(hexStr string) (*btcec.PublicKey, error) {
	pkBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}

	switch len(pkBytes) {
	case 33:
		// Compressed pubkey (02/03 + 32 bytes)
		return btcec.ParsePubKey(pkBytes)
	case 32:
		// X-only pubkey - prepend 0x02 to make it compressed
		compressedBytes := append([]byte{0x02}, pkBytes...)
		return btcec.ParsePubKey(compressedBytes)
	default:
		return nil, fmt.Errorf("invalid pubkey length: expected 32 or 33 bytes, got %d", len(pkBytes))
	}
}

// fetchBabylonParams retrieves staking parameters from the Babylon API
func fetchBabylonParams(apiURL string) (*BabylonParams, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Babylon params: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var params BabylonParams
	if err := json.Unmarshal(body, &params); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &params, nil
}

type cliParams struct {
	stakerPkHex    string
	fpPkHex        string
	stakingAmount  int64
	stakingTime    int
	useTestnet     bool
	apiURL         string
}

func setupUsage() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Babylon BTC Staking Transaction Builder\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", "babylon_txcheck")
		fmt.Fprintf(flag.CommandLine.Output(), "Required Parameters:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -staker-pk string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Staker public key (hex: 64 chars x-only or 66 chars compressed)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -fp-pk string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Finality provider public key (hex: 64 chars x-only or 66 chars compressed)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -amount int\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Staking amount in satoshis\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nOptional Parameters:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -time int\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Staking time in blocks (default: use API minimum)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -testnet\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Use testnet parameters (default: false, uses mainnet)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -api string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Babylon API endpoint (default: https://staking-api.babylonlabs.io/v2/network-info)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nExample:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -staker-pk <key> -fp-pk <key> -amount 1000000\n", "babylon_txcheck")
	}
}

func parseFlags() *cliParams {
	setupUsage()

	params := &cliParams{}
	flag.StringVar(&params.stakerPkHex, "staker-pk", "", "")
	flag.StringVar(&params.fpPkHex, "fp-pk", "", "")
	flag.Int64Var(&params.stakingAmount, "amount", 0, "")
	flag.IntVar(&params.stakingTime, "time", 0, "")
	flag.BoolVar(&params.useTestnet, "testnet", false, "")
	flag.StringVar(&params.apiURL, "api", "https://staking-api.babylonlabs.io/v2/network-info", "")
	flag.Parse()

	return params
}

func validateRequiredParams(params *cliParams) {
	if params.stakerPkHex == "" {
		log.Fatalf("Staker public key (-staker-pk) is required")
	}
	if params.fpPkHex == "" {
		log.Fatalf("Finality provider public key (-fp-pk) is required")
	}
	if params.stakingAmount == 0 {
		log.Fatalf("Staking amount (-amount) is required")
	}
}

func fetchAndValidateParams(params *cliParams) (*BabylonParams, int) {
	fmt.Printf("Fetching parameters from Babylon API: %s\n", params.apiURL)
	babylonParams, err := fetchBabylonParams(params.apiURL)
	if err != nil {
		log.Fatalf("Failed to fetch Babylon parameters: %v", err)
	}

	latest := babylonParams.GetLatestParams()
	if latest == nil {
		log.Fatalf("No parameter versions found in API response")
	}

	fmt.Printf("✓ Successfully fetched parameters (Version: %d, Covenant quorum: %d/%d)\n",
		latest.Version, latest.CovenantQuorum, len(latest.CovenantPks))
	fmt.Println()

	finalTime := params.stakingTime
	if finalTime == 0 {
		finalTime = int(latest.MinStakingTimeBlocks)
		fmt.Printf("Using API minimum staking time: %d blocks\n", finalTime)
	}

	// Validate against API limits
	if params.stakingAmount < latest.MinStakingValueSat {
		log.Fatalf("Staking amount %d is below minimum %d", params.stakingAmount, latest.MinStakingValueSat)
	}
	if params.stakingAmount > latest.MaxStakingValueSat {
		log.Fatalf("Staking amount %d exceeds maximum %d", params.stakingAmount, latest.MaxStakingValueSat)
	}
	if uint32(finalTime) < latest.MinStakingTimeBlocks {
		log.Fatalf("Staking time %d is below minimum %d", finalTime, latest.MinStakingTimeBlocks)
	}
	if uint32(finalTime) > latest.MaxStakingTimeBlocks {
		log.Fatalf("Staking time %d exceeds maximum %d", finalTime, latest.MaxStakingTimeBlocks)
	}

	return babylonParams, finalTime
}

func displayParams(amount int64, finalTime int, net *chaincfg.Params) {
	unlockDays := float64(finalTime) * 10 / 60 / 24
	unlockWeeks := unlockDays / 7
	unlockMonths := unlockDays / 30.44

	fmt.Printf("Staking Amount: %d satoshis\n", amount)
	fmt.Printf("Staking Time: %d blocks (≈ %.1f days / %.1f weeks / %.1f months)\n", finalTime, unlockDays, unlockWeeks, unlockMonths)

	if net == &chaincfg.TestNet3Params {
		fmt.Println("Network: Testnet")
	} else {
		fmt.Println("Network: Mainnet")
	}
	fmt.Println()
}

func parseCovenantKeys(covenantPksHex []string) []*btcec.PublicKey {
	if len(covenantPksHex) == 0 {
		log.Fatalf("No covenant public keys found in API response")
	}

	covenantPubKeys := make([]*btcec.PublicKey, 0, len(covenantPksHex))
	for i, covenantPkHex := range covenantPksHex {
		covenantPk, err := parsePubKey(covenantPkHex)
		if err != nil {
			log.Fatalf("Failed to parse covenant public key %d: %v", i, err)
		}
		covenantPubKeys = append(covenantPubKeys, covenantPk)
	}
	return covenantPubKeys
}

func displayKeys(stakerPubKey, fpPubKey *btcec.PublicKey, covenantPubKeys []*btcec.PublicKey, quorum uint32) {
	fmt.Println("Keys Summary:")
	fmt.Printf("  Staker PK: %s\n", hex.EncodeToString(stakerPubKey.SerializeCompressed()[1:]))
	fmt.Printf("  Finality Provider PK: %s\n", hex.EncodeToString(fpPubKey.SerializeCompressed()[1:]))
	fmt.Printf("  Covenant Committee: %d keys (quorum: %d)\n", len(covenantPubKeys), quorum)
	for i, pk := range covenantPubKeys {
		fmt.Printf("    [%d] %s\n", i+1, hex.EncodeToString(pk.SerializeCompressed()[1:]))
	}
	fmt.Println()
}

func displayUnbondingOutput(stakerPubKey, fpPubKey *btcec.PublicKey, covenantPubKeys []*btcec.PublicKey, covenantQuorum uint32, unbondingTime uint32, stakingAmount int64, net *chaincfg.Params) {
	// Build unbonding output
	unbondingInfo, err := btcstaking.BuildUnbondingInfo(
		stakerPubKey,
		[]*btcec.PublicKey{fpPubKey},
		covenantPubKeys,
		covenantQuorum,
		uint16(unbondingTime),
		btcutil.Amount(stakingAmount),
		net,
	)
	if err != nil {
		log.Fatalf("Failed to build unbonding info: %v", err)
	}

	// Get spending path information
	timeLockSpendInfo, err := unbondingInfo.TimeLockPathSpendInfo()
	if err != nil {
		log.Fatalf("Failed to get unbonding timelock spend info: %v", err)
	}
	slashingSpendInfo, err := unbondingInfo.SlashingPathSpendInfo()
	if err != nil {
		log.Fatalf("Failed to get unbonding slashing spend info: %v", err)
	}

	unbondingAddress, err := btcutil.NewAddressTaproot(
		unbondingInfo.UnbondingOutput.PkScript[2:],
		net,
	)
	if err != nil {
		log.Fatalf("Failed to create unbonding Taproot address: %v", err)
	}

	fmt.Println("Spending Paths:")
	fmt.Println("  1. TimeLock Path (normal unbonding after unbonding time):")
	timeLockCBBytes, _ := timeLockSpendInfo.ControlBlock.ToBytes()
	fmt.Printf("     Script: %s\n", hex.EncodeToString(timeLockSpendInfo.RevealedLeaf.Script))
	fmt.Printf("     Control Block: %s\n", hex.EncodeToString(timeLockCBBytes))
	fmt.Println()
	fmt.Println("  2. Slashing Path (slashing with FP and covenant cooperation):")
	slashingCBBytes, _ := slashingSpendInfo.ControlBlock.ToBytes()
	fmt.Printf("     Script: %s\n", hex.EncodeToString(slashingSpendInfo.RevealedLeaf.Script))
	fmt.Printf("     Control Block: %s\n", hex.EncodeToString(slashingCBBytes))
	fmt.Println()

	fmt.Println("Unbonding Output Information:")
	fmt.Printf("  Value: %d satoshis\n", unbondingInfo.UnbondingOutput.Value)
	fmt.Printf("  Taproot Address: %s\n", unbondingAddress.EncodeAddress())
	fmt.Printf("  PkScript (hex): %s\n", hex.EncodeToString(unbondingInfo.UnbondingOutput.PkScript))
	fmt.Printf("  PkScript Length: %d bytes\n", len(unbondingInfo.UnbondingOutput.PkScript))
	fmt.Println()
	fmt.Println("  Note: This output is used as input for one of the slashing transaction, the unbonding slashing transaction")
	fmt.Println()
}

func displaySlashingOutputs(stakerPubKey *btcec.PublicKey, unbondingTime uint32, net *chaincfg.Params) {
	// Build the slashing change output (where change returns to staker after timelock)
	slashingChangeOutput, err := btcstaking.BuildRelativeTimelockTaprootScript(
		stakerPubKey,
		uint16(unbondingTime),
		net,
	)
	if err != nil {
		log.Fatalf("Failed to build slashing change output: %v", err)
	}

	slashChangeCBBytes, _ := slashingChangeOutput.SpendInfo.ControlBlock.ToBytes()

	fmt.Println("Slashing Change Output Script:")
	fmt.Printf("  Script: %s\n", hex.EncodeToString(slashingChangeOutput.SpendInfo.RevealedLeaf.Script))
	fmt.Printf("  Control Block: %s\n", hex.EncodeToString(slashChangeCBBytes))
	fmt.Printf("  Timelock: %d blocks (unbonding time)\n", slashingChangeOutput.LockTime)
	fmt.Println()

	fmt.Println("Slashing Transaction Outputs:")
	fmt.Println("  Output 1 - Slashing Amount:")
	fmt.Println("     Destination: Burn address (specified in Babylon Genesis parameters)")
	fmt.Println("     Note: This output receives the slashed portion of staked funds")
	fmt.Println()
	fmt.Println("  Output 2 - Change to Staker (Timelock Protected):")
	fmt.Printf("     Taproot Address: %s\n", slashingChangeOutput.TapAddress.EncodeAddress())
	fmt.Printf("     PkScript (hex): %s\n", hex.EncodeToString(slashingChangeOutput.PkScript))
	fmt.Printf("     PkScript Length: %d bytes\n", len(slashingChangeOutput.PkScript))
	fmt.Println()
	fmt.Println("  Note: Both staking and unbonding slashing transactions use the same change output format")
	fmt.Println()
}

func displayStakingOutput(stakingInfo *btcstaking.StakingInfo, net *chaincfg.Params) {
	// Get spending path information
	timeLockSpendInfo, err := stakingInfo.TimeLockPathSpendInfo()
	if err != nil {
		log.Fatalf("Failed to get timelock spend info: %v", err)
	}
	unbondingSpendInfo, err := stakingInfo.UnbondingPathSpendInfo()
	if err != nil {
		log.Fatalf("Failed to get unbonding spend info: %v", err)
	}
	slashingSpendInfo, err := stakingInfo.SlashingPathSpendInfo()
	if err != nil {
		log.Fatalf("Failed to get slashing spend info: %v", err)
	}

	// Extract Taproot address
	taprootAddress, err := btcutil.NewAddressTaproot(
		stakingInfo.GetPkScript()[2:], // Skip witness version and length
		net,
	)
	if err != nil {
		log.Fatalf("Failed to create Taproot address: %v", err)
	}

	// Display spending paths
	fmt.Println("Spending Paths:")
	fmt.Println("  1. TimeLock Path (normal unbonding after staking time):")
	timeLockCBBytes, _ := timeLockSpendInfo.ControlBlock.ToBytes()
	fmt.Printf("     Script: %s\n", hex.EncodeToString(timeLockSpendInfo.RevealedLeaf.Script))
	fmt.Printf("     Control Block: %s\n", hex.EncodeToString(timeLockCBBytes))
	fmt.Println()

	fmt.Println("  2. Unbonding Path (early unbonding with covenant cooperation):")
	unbondingCBBytes, _ := unbondingSpendInfo.ControlBlock.ToBytes()
	fmt.Printf("     Script: %s\n", hex.EncodeToString(unbondingSpendInfo.RevealedLeaf.Script))
	fmt.Printf("     Control Block: %s\n", hex.EncodeToString(unbondingCBBytes))
	fmt.Println()

	fmt.Println("  3. Slashing Path (slashing with FP and covenant cooperation):")
	slashingCBBytes, _ := slashingSpendInfo.ControlBlock.ToBytes()
	fmt.Printf("     Script: %s\n", hex.EncodeToString(slashingSpendInfo.RevealedLeaf.Script))
	fmt.Printf("     Control Block: %s\n", hex.EncodeToString(slashingCBBytes))
	fmt.Println()

	// Display staking output information
	fmt.Println("Staking Output Information:")
	fmt.Printf("  Value: %d satoshis\n", stakingInfo.StakingOutput.Value)
	fmt.Printf("  Taproot Address: %s\n", taprootAddress.EncodeAddress())
	fmt.Printf("  PkScript (hex): %s\n", hex.EncodeToString(stakingInfo.GetPkScript()))
	fmt.Printf("  PkScript Length: %d bytes\n", len(stakingInfo.GetPkScript()))
	fmt.Println()
}

func main() {
	fmt.Println("=== Babylon-Style BTC Staking Transaction Builder ===")

	// Parse and validate CLI parameters
	params := parseFlags()
	validateRequiredParams(params)

	// Fetch and validate Babylon parameters
	babylonParams, finalTime := fetchAndValidateParams(params)

	// Select network
	var net *chaincfg.Params
	if params.useTestnet {
		net = &chaincfg.TestNet3Params
	} else {
		net = &chaincfg.MainNetParams
	}

	// Display parameters
	displayParams(params.stakingAmount, finalTime, net)

	// Parse public keys
	stakerPubKey, err := parsePubKey(params.stakerPkHex)
	if err != nil {
		log.Fatalf("Failed to parse staker public key: %v", err)
	}
	fpPubKey, err := parsePubKey(params.fpPkHex)
	if err != nil {
		log.Fatalf("Failed to parse finality provider public key: %v", err)
	}
	latest := babylonParams.GetLatestParams()
	covenantPubKeys := parseCovenantKeys(latest.CovenantPks)

	// Display keys
	displayKeys(stakerPubKey, fpPubKey, covenantPubKeys, latest.CovenantQuorum)

	// Build staking info using Babylon's implementation
	stakingInfo, err := btcstaking.BuildStakingInfo(
		stakerPubKey,
		[]*btcec.PublicKey{fpPubKey},
		covenantPubKeys,
		latest.CovenantQuorum,
		uint16(finalTime),
		btcutil.Amount(params.stakingAmount),
		net,
	)
	if err != nil {
		log.Fatalf("Failed to build staking info: %v", err)
	}

	fmt.Println("✓ Successfully built staking output using Babylon's BuildStakingInfo!")
	fmt.Println()

	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("STAKING OUTPUT")
	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
	fmt.Println()

	// Display staking output
	displayStakingOutput(stakingInfo, net)

	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("UNBONDING OUTPUT")
	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
	fmt.Println()

	// Display unbonding output
	displayUnbondingOutput(stakerPubKey, fpPubKey, covenantPubKeys, latest.CovenantQuorum, latest.UnbondingTimeBlocks, params.stakingAmount, net)

	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("SLASHING OUTPUTS")
	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
	fmt.Println()

	// Display slashing outputs
	displaySlashingOutputs(stakerPubKey, latest.UnbondingTimeBlocks, net)

	fmt.Println("=== Success! ===")
	fmt.Println("All staking, unbonding, and slashing outputs calculated successfully.")
}
