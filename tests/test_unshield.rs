// #![cfg(feature = "test-bpf")]

// mod test_shield;
// mod helpers;

use borsh::BorshDeserialize;
use solana_program_test::*;
use solana_sdk::{
    account::Account,
    instruction::{AccountMeta, Instruction},
    pubkey::Pubkey,
    signature::Signer,
    transaction::Transaction,
};
use std::mem;
use solana_bridge::{
    instruction::BridgeInstruction,
    processor::process_instruction,
    state::{IncognitoProxy, MAX_BEACON_ADDRESSES},
};
use spl_token::instruction::approve;

#[tokio::test]
async fn test_unshield_success() {
    // new vault program id
    let program_id = Pubkey::new_unique();
    // new shield maker account
    let shield_maker = Pubkey::new_unique();
    // incognito proxy account
    let incognito_proxy = Pubkey::new_unique();

    let mut test = ProgramTest::new(
        "bridge_solana",
        program_id,
        processor!(process_instruction),
    );

    // limit to track compute unit increase
    test.set_compute_max_units(38_000);

    test.add_account(
        incognito_proxy,
        Account {
            lamports: 5,
            data: vec![0_u8; mem::size_of::<u32>()],
            owner: program_id,
            ..Account::default()
        },
    );

    let (mut banks_client, payer, recent_blockhash) = test.start().await;

    // let mut transaction = Transaction::new_with_payer(
    //     &[
    //         approve(
    //             &spl_token::id(),
    //             &sol_test_reserve.user_collateral_pubkey,
    //             &user_transfer_authority.pubkey(),
    //             &user_accounts_owner.pubkey(),
    //             &[],
    //             SOL_DEPOSIT_AMOUNT_LAMPORTS,
    //         )
    //             .unwrap(),
    //         deposit_obligation_collateral(
    //             spl_token::id(),
    //             SOL_DEPOSIT_AMOUNT_LAMPORTS,
    //             sol_test_reserve.user_collateral_pubkey,
    //             sol_test_reserve.collateral_supply_pubkey,
    //             sol_test_reserve.pubkey,
    //             test_obligation.pubkey,
    //             lending_market.pubkey,
    //             test_obligation.owner,
    //             user_transfer_authority.pubkey(),
    //         ),
    //     ],
    //     Some(&payer.pubkey()),
    // );
}