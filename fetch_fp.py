#!/usr/bin/env python3
"""
Babylon Finality Provider Selector
Fetches finality providers from Babylon API and allows interactive selection.
"""

import requests
import json
import sys
from wcwidth import wcswidth

def fetch_finality_providers():
    """Fetch finality providers from Babylon API"""
    url = "https://babylon.nodes.guru/babylon/btcstaking/v1/finality_providers"

    print("Fetching finality providers from Babylon API...")
    try:
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        data = response.json()
        return data.get('finality_providers', [])
    except requests.exceptions.RequestException as e:
        print(f"Error fetching data: {e}")
        sys.exit(1)

def visual_width(text):
    """Calculate the visual display width of text including emoji"""
    width = wcswidth(text)
    return width if width >= 0 else len(text)

def get_status(fp):
    """Determine the status of a finality provider"""
    slashed_babylon = int(fp.get('slashed_babylon_height', 0))
    slashed_btc = int(fp.get('slashed_btc_height', 0))
    jailed = fp.get('jailed', False)
    soft_deleted = fp.get('soft_deleted', False)

    if slashed_babylon > 0 or slashed_btc > 0:
        return "slashed"
    elif jailed:
        return "jailed"
    elif soft_deleted:
        return "inactive"
    else:
        return "active"

def display_providers(providers, indices=None):
    """Display finality providers in a table format

    Args:
        providers: List of provider dictionaries
        indices: Optional list of original indices to display (keeps original numbering)
    """
    if not providers:
        print("No finality providers found.")
        sys.exit(1)

    # Use original indices if provided, otherwise number sequentially
    if indices is None:
        indices = list(range(1, len(providers) + 1))

    # Column widths (visual width)
    col_num = 5
    col_desc = 35
    col_status = 10
    col_commission = 7
    col_pk = 13  # 10 chars + "..."
    total_width = col_num + col_desc + col_status + col_commission + col_pk + 6  # +6 for spaces (3 after commission)

    print("\n" + "="*total_width)
    header = f"{'#':<{col_num}} {'Description':<{col_desc}} {'Status':<{col_status}} {'Commission':<{col_commission}}   {'BTC PubKey':<{col_pk}}"
    print(header)
    print("="*total_width)

    for idx, fp in zip(indices, providers):
        # Extract relevant fields
        description = fp.get('description', {})
        moniker = description.get('moniker', 'N/A')
        btc_pk = fp.get('btc_pk', 'N/A')
        status = get_status(fp)

        # Format commission as percentage
        commission_str = fp.get('commission', '0')
        try:
            commission_pct = float(commission_str) * 100
            commission_display = f"{commission_pct:.2f}%"
        except (ValueError, TypeError):
            commission_display = "N/A"

        # Abbreviate public key
        pk_display = btc_pk[:10] + "..." if len(btc_pk) > 10 else btc_pk

        # Truncate moniker if visual width is too long
        while visual_width(moniker) > col_desc - 1:
            moniker = moniker[:-4] + "..."

        # Calculate adjusted padding for f-string based on visual width difference
        desc_pad = col_desc - (visual_width(moniker) - len(moniker))
        status_pad = col_status - (visual_width(status) - len(status))

        print(f"{idx:<{col_num}} {moniker:<{desc_pad}} {status:<{status_pad}} {commission_display:>{col_commission}}   {pk_display}")

    print("="*total_width)
    print(f"\nTotal finality providers: {len(providers)}\n")

    return providers

def search_providers(providers, search_term):
    """Filter providers by moniker containing search term (case-insensitive)

    Returns:
        Tuple of (filtered_providers, original_indices)
    """
    search_lower = search_term.lower()
    matches = []
    indices = []
    for i, fp in enumerate(providers, 1):
        if search_lower in fp.get('description', {}).get('moniker', '').lower():
            matches.append(fp)
            indices.append(i)
    return matches, indices

def select_from_filtered(filtered_providers, filtered_indices, original_providers, search_term):
    """Allow selection from filtered results or search again

    Args:
        filtered_providers: List of filtered provider dictionaries
        filtered_indices: List of original indices for the filtered providers
        original_providers: Full list of all providers
        search_term: The search term used for filtering
    """
    while True:
        try:
            prompt = f"Enter number to select, new search term, 'b' for full list, or 'q' to quit: "
            choice = input(prompt).strip()

            if choice.lower() == 'q':
                print("Exiting...")
                sys.exit(0)

            if choice.lower() == 'b':
                return None  # Signal to go back to full list

            # Try to parse as number for selection
            try:
                idx = int(choice)
                # Check if this number is in our filtered indices
                if idx in filtered_indices:
                    # Return the provider at the original index
                    return original_providers[idx - 1]
                else:
                    valid_nums = ', '.join(map(str, filtered_indices))
                    print(f"Please enter one of the displayed numbers: {valid_nums}")
                    continue
            except ValueError:
                # It's a new search term
                new_matches, new_indices = search_providers(original_providers, choice)
                if new_matches:
                    print(f"\nFound {len(new_matches)} match(es) for '{choice}':")
                    display_providers(new_matches, new_indices)
                    return select_from_filtered(new_matches, new_indices, original_providers, choice)
                else:
                    print(f"No providers found matching '{choice}'. Try again.")
        except KeyboardInterrupt:
            print("\nExiting...")
            sys.exit(0)

def select_provider(providers):
    """Allow user to select a finality provider by number or search by text"""
    while True:
        try:
            choice = input("Enter number to select, search term to filter, or 'q' to quit: ").strip()

            if choice.lower() == 'q':
                print("Exiting...")
                sys.exit(0)

            # Try to parse as number first
            try:
                idx = int(choice)
                if 1 <= idx <= len(providers):
                    return providers[idx - 1]
                else:
                    print(f"Please enter a number between 1 and {len(providers)}")
            except ValueError:
                # It's a search term
                matches, indices = search_providers(providers, choice)
                if matches:
                    print(f"\nFound {len(matches)} match(es) for '{choice}':")
                    display_providers(matches, indices)
                    result = select_from_filtered(matches, indices, providers, choice)
                    if result is not None:
                        return result
                    # If None, user chose 'b' to go back - redisplay full list
                    print("\nShowing full list:")
                    display_providers(providers)
                else:
                    print(f"No providers found matching '{choice}'. Try again.")
        except KeyboardInterrupt:
            print("\nExiting...")
            sys.exit(0)

def display_selected_info(provider):
    """Display detailed information about selected provider"""
    description = provider.get('description', {})

    print("\n" + "="*80)
    print("SELECTED FINALITY PROVIDER")
    print("="*80)
    print(f"Moniker:        {description.get('moniker', 'N/A')}")
    print(f"Identity:       {description.get('identity', 'N/A')}")
    print(f"Website:        {description.get('website', 'N/A')}")
    print(f"Security:       {description.get('security_contact', 'N/A')}")
    print(f"Details:        {description.get('details', 'N/A')}")
    print("-"*80)
    print(f"Commission:     {provider.get('commission', 'N/A')}")
    print(f"Address:        {provider.get('addr', 'N/A')}")
    print(f"Jailed:         {provider.get('jailed', False)}")
    print(f"Slashed BBN:    {provider.get('slashed_babylon_height', '0')}")
    print(f"Slashed BTC:    {provider.get('slashed_btc_height', 0)}")
    print("="*80)
    print("\nBTC PUBLIC KEY (hex):")
    print("-"*80)
    btc_pk = provider.get('btc_pk', 'N/A')
    print(btc_pk)
    print("="*80)

def main():
    print("=" * 80)
    print("Babylon Finality Provider Selector")
    print("=" * 80)

    # Fetch finality providers
    providers = fetch_finality_providers()

    # Display in table
    providers = display_providers(providers)

    # Let user select one
    selected = select_provider(providers)

    # Display detailed info
    display_selected_info(selected)

if __name__ == "__main__":
    main()
