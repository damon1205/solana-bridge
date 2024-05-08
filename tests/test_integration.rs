// use solana_client::rpc_client::RpcClient;
// use solana_program::pubkey::Pubkey;
// use solana_sdk::{
//     commitment_config::CommitmentConfig,
//     native_token::LAMPORTS_PER_SOL,
//     signature::{Keypair, Signer},
//     system_instruction
// };
//
// #[test]
// fn test_integration_shield() {
//     let payer = Keypair::from_bytes(&[1,185,72,49,215,81,171,50,85,54,122,53,24,248,3,221,42,85,82,43,128,80,215,127,68,99,172,141,116,237,232,85,185,31,141,73,173,222,173,174,4,212,0,104,157,80,63,147,21,81,140,201,113,76,156,161,154,92,70,67,163,52,219,72]);
//     let pub_key = payer.unwrap().pubkey();
//     let program_id_str = "CE11YS7vdYgRFKvV5pfRJknjMxpz9rusnpeYFDMJwKDd".as_ref();
//     let vault_program = Pubkey::new_from_array(program_id_str);
//     let rpc_url = String::from("https://api.devnet.solana.com");
//     let client = RpcClient::new_with_commitment(rpc_url, CommitmentConfig::confirmed());
//     match client.request_airdrop(&pub_key, LAMPORTS_PER_SOL) {
//         Ok(sig) => loop {
//             if let Ok(confirmed) = client.confirm_transaction(&sig) {
//                 if confirmed {
//                     println!("Transaction: {} Status: {}", sig, confirmed);
//                     break;
//                 }
//             }
//         },
//         Err(_) => println!("Error requesting airdrop"),
//     };
//
//     let account = Keypair::new();
//     let space =
//     println!("new account address = {}", account.pubkey());
//
//     // init incognito proxy account
//     let create_incoggnito_proxy = system_instruction::create_account(
//         &pub_key,
//         &account.pubkey(),
//     rent,
//     space as u64,
//     &wallet.pubkey(),
//     );
//
// }