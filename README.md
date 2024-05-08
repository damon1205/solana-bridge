# SOLANA <> INCOGNITO BRIDGE
Bridge program to bring privacy to Solana ecosystem.

# PROGRAM GUIDE

### Prerequisites
- Rust ([link](https://doc.rust-lang.org/cargo/getting-started/installation.html))
- Solana client tool ([link](https://docs.solana.com/cli/install-solana-cli-tools)) 

### Build program:

```sh
cargo build-bpf 
```

### Deploy program:

```sh
solna program deploy $PROGRAM_PATH
```

## Features

- Shield / UnShield between Solana and Incognito chain
- Provide privacy for dapp on solana

## Clients

- Golang 
- Typescript

# INSTRUCTIONS

## Init incognito proxy instruction

    ///   Initializes a new Incognito proxy account.
    ///
    ///   0. `[]` $SYSVAR_RENT_PUBKEY to check account rent exempt
    ///   1. `[writable]` Incognito proxy account
    ///   2. `[writable]` Vault account
    InitBeacon {
        /// beacon info
        init_beacon_info: IncognitoProxy,
    }

## Shield instruction

    ///   Request new shield to move token from Solana -> Incognito.
    ///
    ///   0. `[writable]` Token account to make shield request
    ///   1. `[writable]` Vault token account to receive token from asker
    ///   2. `[]` Incognito proxy which stores beacon list and bump seed to retrieve vault token account
    ///   3. `[signer]` Shield maker address
    ///   4. `[]` Spl Token program id
    Shield {
        /// amount to shield
        amount: u64,
        /// destination in privacy layer
        inc_address: [u8; 148],
    }

## Unshield instruction

    ///   Request new unshield to move token from Incognito -> Solana.
    ///
    ///   0. `[writable]` Vault token account to transfer tokens to unshield maker
    ///   1. `[]` Unshield maker address
    ///   2. `[]` $vault_authority derived from `create_program_address(&[incognito proxy account])`
    ///   3. `[writable]` pda account to mark burn id already used
    ///   4. `[]` Incognito proxy which stores beacon list and bump seed to retrieve vault token account
    ///   5. `[]` Spl Token program id
    ///   6. `[writable]` Associated token account of unshield maker
    ///   7. `[signer]` Fee payer for pda account which stores tx id
    ///   8. `[]` System program to create pda account
    UnShield {
        /// unshield info
        unshield_info: UnshieldRequest,
    }

## Dapp interaction instruction

    /// Generic instruction to allow vault interact with any dapp on Solana.
    ///
    /// 0. `[signer]` Signer account
    /// ...
    DappInteraction {
        /// beacon info
        dapp_request: DappRequest,
    }

## Withdraw request instruction

    ///   Withdraw token from signer token account to shield back to Incognito.
    ///
    ///   0. `[writable]` Signer token account to withdraw
    ///   1. `[writable]` Vault token account to receive token from signer
    ///   2. `[]` Incognito proxy which stores beacon list and bump seed to retrieve vault token account
    ///   3. `[signer]` signer request address
    ///   4. `[]` $signer_authority derived from `create_program_address(&[signer account])`
    ///   5. `[]` Spl Token program id
    WithdrawRequest {
        /// withdraw request
        amount: u64,
        inc_address: [u8; 148],
    }
