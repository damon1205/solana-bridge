// #![cfg(feature = "test-bpf")]
mod helpers;

use solana_program_test::*;
use solana_sdk::{
    account::Account,
    instruction::{AccountMeta, Instruction},
    pubkey::Pubkey,
    signature::{Keypair, Signer},
    transaction::Transaction,
    secp256k1_recover::Secp256k1Pubkey,
};
use solana_bridge::{
    instruction::BridgeInstruction,
    processor::process_instruction,
    state::{IncognitoProxy, MAX_BEACON_ADDRESSES},
};
use spl_token::{
    instruction::approve,
    state::{Account as Token, AccountState, Mint},
};
use solana_bridge::instruction::shield;

use crate::helpers::{add_packable_account, get_token_balance};

#[tokio::test]
async fn test_shield_success() {
    let pubkey = Pubkey::new(&[0x3b,0xfb,0x14,0x08,0xa9,0x29,0x0e,0x7e,0xd9,0xca,0xd0,0x85,0xf5,0xc2,0x3e,0xf9,0x90,0x21,0x72,0xbd,0x45,0x08,0xcb,0xec,0x21,0xca,0x8b,0xfc,0x16,0xb6,0x54,0xa6]);
    println!("======= {}", pubkey);

    // new shield maker account
    let shield_maker = Keypair::new();
    // shield maker token account
    let shield_maker_token_account = Pubkey::new_unique();
    // new vault program id
    let program_id = Pubkey::new_unique();
    // incognito proxy account
    let incognito_proxy = Pubkey::new_unique();
    // vault token account
    let vault_token_account = Pubkey::new_unique();
    // mint pub key
    let token_mint_pub_key = Pubkey::new_unique();
    // token program spl_token::id()
    // new vault account id
    let vault_account_id = Pubkey::new_unique();

    let deposit_amount: u64 = 100_0000;
    let (incognito_proxy_authority_key, bump_seed) =
        Pubkey::find_program_address(&[incognito_proxy.as_ref()], &program_id);

    let authority_signer_seeds = &[
        incognito_proxy.as_ref(),
        &[bump_seed],
    ];
    let incognito_proxy_authority_key_gen = Pubkey::create_program_address(authority_signer_seeds, &program_id).unwrap();

    println!("generated incognito authority key {}, {}", incognito_proxy_authority_key.to_string(), incognito_proxy_authority_key_gen.to_string());

    let mut test = ProgramTest::new(
        "bridge_solana",
        program_id,
        processor!(process_instruction),
    );

    // limit to track compute unit increase
    test.set_compute_max_units(38_000);

    println!("bump seeed in test {}", bump_seed);
    add_packable_account(
        &mut test,
        incognito_proxy,
        u32::MAX as u64,
        &IncognitoProxy::new(IncognitoProxy {
            is_initialized: true,
            bump_seed,
            vault: vault_account_id,
            beacons: Vec::new(), // todo add beacons
        }),
        &program_id,
    );

    // init shield maker token account
     add_packable_account(
        &mut test,
        shield_maker_token_account,
        u32::MAX as u64,
        &Token {
            mint: token_mint_pub_key,
            owner: shield_maker.pubkey(),
            amount: deposit_amount,
            state: AccountState::Initialized,
            ..Token::default()
        },
        &spl_token::id(),
    );

    // init vault token account
    add_packable_account(
        &mut test,
        vault_token_account,
        u32::MAX as u64,
        &Token {
            mint: token_mint_pub_key,
            owner: incognito_proxy_authority_key,
            amount: 0,
            state: AccountState::Initialized,
            ..Token::default()
        },
        &spl_token::id(),
    );

    let (mut banks_client, payer, recent_blockhash) = test.start().await;

    let inital_shield_maker_account =
        get_token_balance(&mut banks_client, shield_maker_token_account).await;
    let initial_vault_token_account =
        get_token_balance(&mut banks_client, vault_token_account).await;

    println!("amount shield maker and vault account before {}, {}", initial_vault_token_account, inital_shield_maker_account);

    let mut transaction = Transaction::new_with_payer(
        &[
            shield(
                program_id,
                deposit_amount,
                shield_maker_token_account,
                vault_token_account,
                incognito_proxy,
                shield_maker.pubkey(),
                &[1; 148],
            ),
        ],
        Some(&payer.pubkey()),
    );

    transaction.sign(
        &vec![&payer, &shield_maker],
        recent_blockhash,
    );
    assert!(banks_client.process_transaction(transaction).await.is_ok());

    let after_shield_maker_account =
        get_token_balance(&mut banks_client, shield_maker_token_account).await;
    let after_vault_token_account =
        get_token_balance(&mut banks_client, vault_token_account).await;
    assert_eq!(
        after_shield_maker_account,
        inital_shield_maker_account - deposit_amount
    );
    assert_eq!(
        after_vault_token_account,
        initial_vault_token_account + deposit_amount
    );
}