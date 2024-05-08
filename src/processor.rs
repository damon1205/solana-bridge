use solana_program::{
    account_info::{next_account_info, AccountInfo},
    entrypoint::ProgramResult,
    msg,
    program::{invoke, invoke_signed},
    program_error::ProgramError,
    program_pack::{IsInitialized, Pack},
    pubkey::Pubkey,
    sysvar::{rent::Rent, Sysvar},
    instruction::{Instruction, AccountMeta},
    secp256k1_recover::secp256k1_recover,
    keccak::hash,
    borsh::try_from_slice_unchecked,
    system_instruction,
};
use std::{
    str,
};
use borsh::{BorshSerialize};
use spl_token::state::Account as TokenAccount;
use arrayref::{array_refs, array_ref};
use crate::{error::BridgeError, instruction::BridgeInstruction, state::{UnshieldRequest, IncognitoProxy, PDABurnId}};
use crate::state::{DappRequest};
use spl_associated_token_account::{get_associated_token_address};

const LEN: usize = 1 + 1 + 32 + 32 + 32 + 32; // ignore last 32 bytes in instruction

pub fn process_instruction(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        instruction_data: &[u8],
) -> ProgramResult {
    let instruction = BridgeInstruction::unpack(instruction_data)?;

    match instruction {
        BridgeInstruction::Shield { amount, inc_address } => {
            msg!("Instruction: Shield");
            process_shield(accounts, amount, inc_address, program_id)
        }
        BridgeInstruction::UnShield { unshield_info } => {
            msg!("Instruction: Unshield");
            process_unshield(accounts, unshield_info, program_id)
        }
        BridgeInstruction::InitBeacon { init_beacon_info } => {
            msg!("Instruction: Init beacon list");
            process_init_beacon(accounts, init_beacon_info, program_id)
        }
        BridgeInstruction::DappInteraction {dapp_request} => {
            msg!("Instruction: Dapp interaction");
            process_dapp_interaction(accounts, dapp_request, program_id)
        }
        BridgeInstruction::WithdrawRequest{ amount, inc_address } => {
            msg!("Instruction: Withdraw Request");
            process_withdraw_request(accounts, amount, inc_address, program_id)
        }
    }
}

fn process_shield(
    accounts: &[AccountInfo],
    amount: u64,
    inc_address: [u8; 148],
    program_id: &Pubkey,
) -> ProgramResult {
    let account_info_iter = &mut accounts.iter();
    let shield_maker_token_account = next_account_info(account_info_iter)?;
    let vault_token_account = next_account_info(account_info_iter)?;
    let incognito_proxy = next_account_info(account_info_iter)?;
    let shied_maker = next_account_info(account_info_iter)?;
    if !shied_maker.is_signer {
        return Err(ProgramError::MissingRequiredSignature);
    }
    let token_program = next_account_info(account_info_iter)?;

    if incognito_proxy.owner != program_id {
        msg!("Invalid incognito proxy");
        return Err(ProgramError::IncorrectProgramId);
    }

    if *vault_token_account.owner != spl_token::id() {
        msg!("PDABurnId token account must be owned by spl token");
        return Err(ProgramError::IncorrectProgramId);
    }

    let token_id = _verify_vault_token_account(
        incognito_proxy.clone(),
        vault_token_account.clone(),
        program_id.clone())?;

    spl_token_transfer(TokenTransferParams {
        source: shield_maker_token_account.clone(),
        destination: vault_token_account.clone(),
        amount,
        authority: shied_maker.clone(),
        authority_signer_seeds: &[],
        token_program: token_program.clone(),
    })?;
    msg!("Issue pToken to incognitoproxy,address,token,amount:{},{},{},{}",
        incognito_proxy.key,str::from_utf8(&inc_address[..]).unwrap(), token_id, amount);
    Ok(())
}

/// [x] declare vars
/// [x] extract info from input params
/// [x] verify beacon signatures
/// [x] verify instruction merkle tree
/// [x] detect unshield sol and transfer sol directly to user account
/// [x] store unshield tx id
/// [x] transfer token back to user

// add logic to proccess unshield for users
fn process_unshield(
    accounts: &[AccountInfo],
    unshield_info: UnshieldRequest,
    program_id: &Pubkey,
) -> ProgramResult {
    let account_info_iter = &mut accounts.iter();
    let vault_token_account = next_account_info(account_info_iter)?;
    let unshield_maker = next_account_info(account_info_iter)?;
    let vault_authority_account = next_account_info(account_info_iter)?;
    let pda_account = next_account_info(account_info_iter)?;
    let incognito_proxy = next_account_info(account_info_iter)?;
    let token_program = next_account_info(account_info_iter)?;
    let unshield_token_account = next_account_info(account_info_iter)?;
    let authority_account = next_account_info(account_info_iter)?;
    let system_program = next_account_info(account_info_iter)?;
    let incognito_proxy_info = IncognitoProxy::unpack_unchecked(&incognito_proxy.data.borrow())?;
    if !incognito_proxy_info.is_initialized() {
        return Err(BridgeError::BeaconsUnInitialized.into());
    }

    if !authority_account.is_signer {
        return Err(BridgeError::InvalidAuthorityAccount.into())
    }

    if incognito_proxy.owner != program_id {
        msg!("Invalid incognito proxy");
        return Err(ProgramError::IncorrectProgramId);
    }

    // extract data from input
    let inst = unshield_info.inst;
    if inst.len() < LEN {
        msg!("Invalid instruction input");
        return Err(BridgeError::InvalidBeaconInstruction.into());
    }
    let inst_ = array_ref![inst, 0, LEN];
    #[allow(clippy::ptr_offset_with_cast)]
    let (
        meta_type,
        shard_id,
        token,
        receiver_key,
        _,
        unshield_amount,
        tx_id, // todo: store this data
    ) = array_refs![
        inst_,
        1,
        1,
        32,
        32,
        24,
        8,
        32
    ];
    let meta_type = u8::from_le_bytes(*meta_type);
    let shard_id = u8::from_le_bytes(*shard_id);
    let token_key = Pubkey::new(token);
    let receiver_key = Pubkey::new(receiver_key);
    let unshield_amount_u64 = u64::from_be_bytes(*unshield_amount);

    // validate metatype and key provided
    if (meta_type != 157 && meta_type != 158) || shard_id != 1 {
        msg!("Invalid beacon instruction metatype {}, {}", meta_type, shard_id);
        return Err(BridgeError::InvalidKeysInInstruction.into());
    }

    // verify vault token account
    let token_id = _verify_vault_token_account(
        incognito_proxy.clone(),
        vault_token_account.clone(),
        program_id.clone())?;
    if token_key != token_id {
        msg!("Token key and key provided not match {}, {}", token_key, token_id);
        return Err(BridgeError::InvalidKeysInInstruction.into());
    }

    if receiver_key != *unshield_maker.key {
        msg!("Receive key and key provided not match {}, {}", receiver_key, *unshield_maker.key);
        return Err(BridgeError::InvalidKeysInInstruction.into());
    }

    // verify beacon signature
    if unshield_info.indexes.len() != unshield_info.signatures.len() {
        msg!("Invalid instruction provided, length of indexes and signatures not match");
        return Err(BridgeError::InvalidBeaconInstruction.into());
    }

    if unshield_info.signatures.len() <= incognito_proxy_info.beacons.len() * 2 / 3 {
        msg!("Invalid instruction input");
        return Err(BridgeError::InvalidNumberOfSignature.into());
    }

    let mut blk_data_bytes = unshield_info.blk_data.to_vec();
    blk_data_bytes.extend_from_slice(&unshield_info.inst_root);
    // Get double block hash from instRoot and other data
    let blk = hash(&hash(&blk_data_bytes[..]).to_bytes());

    for i in 0..unshield_info.indexes.len() {
        let s_r_v = unshield_info.signatures[i];
        let (s_r, v) = s_r_v.split_at(64);
        if v.len() != 1 {
            msg!("Invalid signature v input");
            return Err(BridgeError::InvalidBeaconSignature.into());
        }
        let beacon_key_from_signature_result = secp256k1_recover(
            &blk.to_bytes()[..],
            v[0],
            s_r,
        ).unwrap();
        let index_beacon = unshield_info.indexes[i];
        let beacon_key = incognito_proxy_info.beacons[index_beacon as usize];
        if beacon_key_from_signature_result != beacon_key {
            return Err(BridgeError::InvalidBeaconSignature.into());
        }
    }

    // append block height to instruction
    let height_vec = append_at_top(unshield_info.height);
    let mut inst_vec = inst.to_vec();
    inst_vec.extend_from_slice(&height_vec);
    let inst_hash = hash(&inst_vec[..]);
    if !instruction_in_merkle_tree(
        &inst_hash.to_bytes(),
        &unshield_info.inst_root,
        &unshield_info.inst_paths,
        &unshield_info.inst_path_is_lefts
    ) {
        msg!("Invalid instruction root");
        return Err(BridgeError::InvalidBeaconMerkleTree.into());
    }
    // verify pda generated from tx burn id not initialized
    _process_tx_burn_id(pda_account, incognito_proxy, authority_account, system_program, program_id, tx_id)?;

    // prepare to transfer token to user
    let authority_signer_seeds = &[
        incognito_proxy.key.as_ref(),
        &[incognito_proxy_info.bump_seed],
    ];

    // transfer token
    spl_token_transfer(TokenTransferParams {
        source: vault_token_account.clone(),
        destination: unshield_token_account.clone(),
        amount: unshield_amount_u64,
        authority: vault_authority_account.clone(),
        authority_signer_seeds,
        token_program: token_program.clone(),
    })?;

    match meta_type {
        157 => {
            let is_wsol = token_key == spl_token::native_mint::id();
            if !is_wsol {
                let unshield_maker_associated_acc = get_associated_token_address(
                    &unshield_maker.key,
                    &token_key
                );

                if unshield_maker_associated_acc != *unshield_token_account.key {
                    msg!("unshield maker's associated token account and key provided not match {}, {}", unshield_maker_associated_acc, *unshield_token_account.key);
                    return Err(BridgeError::InvalidKeysInInstruction.into());
                }
            } else {
                // handle native token
                if *vault_token_account.key == *unshield_token_account.key {
                    msg!("Invalid sender and receiver in unshield request");
                    return Err(BridgeError::InvalidTransferTokenData.into());
                }
                // close account
                spl_close_token_acc(TokenCloseParams {
                    account: unshield_token_account.clone(),
                    destination: unshield_maker.clone(),
                    authority: vault_authority_account.clone(),
                    authority_signer_seeds,
                    token_program: token_program.clone(),
                })?;
            }
        },
        158 => {
            let (pda, _) = Pubkey::find_program_address(
                &[unshield_maker.key.as_ref()],
                program_id
            );
            let pda_associated_token_acc = get_associated_token_address(&pda, &token_key);
            if pda_associated_token_acc != *unshield_token_account.key {
                msg!("pda's associated token account and key provided not match {}, {}", pda_associated_token_acc, *unshield_token_account.key);
                return Err(BridgeError::InvalidKeysInInstruction.into());
            }
        },
        _ => {
            return Err(BridgeError::InvalidMetaType.into());
        }
    }

    Ok(())
}

// add logic to proccess init beacon list
fn process_init_beacon(
    accounts: &[AccountInfo],
    init_beacon_info: IncognitoProxy,
    program_id: &Pubkey,
) -> ProgramResult {
    let account_info_iter = &mut accounts.iter();
    let rent = &Rent::from_account_info(next_account_info(account_info_iter)?)?;
    let incognito_proxy = next_account_info(account_info_iter)?;

    assert_rent_exempt(rent, incognito_proxy)?;
    let mut incognito_proxy_info = assert_uninitialized::<IncognitoProxy>(incognito_proxy)?;
    if incognito_proxy.owner != program_id {
        msg!("Invalid incognito proxy");
        return Err(ProgramError::IncorrectProgramId);
    }

    incognito_proxy_info.is_initialized = init_beacon_info.is_initialized;
    incognito_proxy_info.bump_seed = init_beacon_info.bump_seed;
    incognito_proxy_info.beacons = init_beacon_info.beacons;
    IncognitoProxy::pack(incognito_proxy_info, &mut incognito_proxy.data.borrow_mut())?;

    Ok(())
}

fn process_withdraw_request(
    accounts: &[AccountInfo],
    amount: u64,
    inc_address: [u8; 148],
    program_id: &Pubkey,
) -> ProgramResult {
    let account_info_iter = &mut accounts.iter();
    let signer_token_account = next_account_info(account_info_iter)?;
    let vault_token_account = next_account_info(account_info_iter)?;
    let incognito_proxy = next_account_info(account_info_iter)?;
    let signer = next_account_info(account_info_iter)?;
    if !signer.is_signer {
        return Err(BridgeError::InvalidSigner.into());
    }
    let signer_authority_token = next_account_info(account_info_iter)?;
    let token_program = next_account_info(account_info_iter)?;

    // verify vault token account
    let token_id = _verify_vault_token_account(
        incognito_proxy.clone(),
        vault_token_account.clone(),
        program_id.clone())?;

    let (pda, bump) = Pubkey::find_program_address(
        &[signer.key.as_ref()],
        program_id
    );

    if pda != *signer_authority_token.key {
        return Err(BridgeError::InvalidSignerTokenAuth.into());
    }

    let authority_signer_seeds = &[
        signer.key.as_ref(),
        &[bump],
    ];

    spl_token_transfer(TokenTransferParams {
        source: signer_token_account.clone(),
        destination: vault_token_account.clone(),
        amount,
        authority: signer_authority_token.clone(),
        authority_signer_seeds,
        token_program: token_program.clone(),
    })?;

    msg!("Issue pToken to incognitoproxy,address,token,amount:{},{},{},{}", incognito_proxy.key,str::from_utf8(&inc_address[..]).unwrap(), token_id, amount);

    Ok(())
}

fn process_dapp_interaction(
    accounts: &[AccountInfo],
    dapp_request: DappRequest,
    program_id: &Pubkey,
) -> ProgramResult {
    let account_info_iter = &mut accounts.iter();
    let signer = next_account_info(account_info_iter)?;
    if !signer.is_signer {
        return Err(BridgeError::InvalidSigner.into());
    }
    let (_, bump) = Pubkey::find_program_address(
        &[signer.key.as_ref()],
        program_id
    );

    let authority_signer_seeds = &[
        signer.key.as_ref(),
        &[bump],
    ];

    let mut accounts_invoke = Vec::with_capacity(dapp_request.num_acc as usize + 1);
    let mut accounts_info = Vec::with_capacity(dapp_request.num_acc as usize + 1);
    for i in 0..dapp_request.num_acc {
        let next_acc: &AccountInfo = next_account_info(account_info_iter)?;
        accounts_info.push(next_acc.clone());
        let is_signer = next_acc.is_signer || dapp_request.sign_index == i;
        if !next_acc.is_writable {
            accounts_invoke.push(AccountMeta::new_readonly(*next_acc.key, is_signer));
        } else {
            accounts_invoke.push(AccountMeta::new(*next_acc.key, is_signer));
        }
    }
    let program_dest = next_account_info(account_info_iter)?;

    let new_inst = Instruction {
        program_id: *program_dest.key,
        accounts: accounts_invoke.to_vec(),
        data: dapp_request.inst,
    };
    invoke_signed(
        &new_inst,
        &accounts_info[..],
        &[authority_signer_seeds],
    ).unwrap();

    Ok(())
}

fn _process_tx_burn_id<'a>(
    pda_account: &AccountInfo<'a>,
    incognito_proxy: &AccountInfo<'a>,
    authority_account: &AccountInfo<'a>,
    system_program: &AccountInfo<'a>,
    program_id: &Pubkey, 
    txid: &[u8; 32],
) -> ProgramResult {
    // verify vault pda account
    let (pda, bump) = Pubkey::find_program_address(
        &[
            incognito_proxy.key.as_ref(),
            txid.as_ref(),
        ],
        program_id
    );
    if pda != *pda_account.key {
        msg!("Mismatch pda generated and pda provided {}, {}", pda, pda_account.key);
        return Err(BridgeError::InvalidPDAAccount.into());
    }
    if !pda_account.data_is_empty() || !pda_account.is_writable {
        msg!("Pda account is not writable or existed");
        return Err(BridgeError::InvalidPDAAccount.into());
    }

    let rent = Rent::get()?;
    let rent_lamports = rent.minimum_balance(PDABurnId::LEN);

    let create_map_ix = &system_instruction::create_account(
        authority_account.key,
        pda_account.key,
        rent_lamports,
        PDABurnId::LEN.try_into().unwrap(),
        program_id
    );

    invoke_signed(
        create_map_ix,
        &[
            authority_account.clone(),
            pda_account.clone(),
            system_program.clone()
        ],
        &[&[
            incognito_proxy.key.as_ref(),
            txid.as_ref(),
            &[bump]
        ]]
    )?;
    
    let mut pda_tx_burn = try_from_slice_unchecked::<PDABurnId>(&pda_account.try_borrow_data()?)?;
    pda_tx_burn.is_initialized = true;
    pda_tx_burn.serialize(&mut &mut pda_account.data.borrow_mut()[..])?;

    Ok(())
}

fn _verify_vault_token_account(incognito_proxy: AccountInfo, vault_token_account: AccountInfo, program_id: Pubkey) -> Result<Pubkey, ProgramError> {
    let vault_token_account_info = TokenAccount::unpack(&vault_token_account.try_borrow_data()?)?;
    let incognito_proxy_info = IncognitoProxy::unpack(&incognito_proxy.try_borrow_data()?)?;
    let authority_signer_seeds = &[
        incognito_proxy.key.as_ref(),
        &[incognito_proxy_info.bump_seed],
    ];
    let vault_authority_pubkey =
        Pubkey::create_program_address(authority_signer_seeds, &program_id)?;

    let incognio_proxy_associated_acc = get_associated_token_address(
        &vault_authority_pubkey,
        &vault_token_account_info.mint
    );

    if incognio_proxy_associated_acc != *vault_token_account.key {
        msg!("Only incognito proxy account will be accepted");
        return Err(ProgramError::IncorrectProgramId);
    }

    Ok(vault_token_account_info.mint)
}

// check rent exempt
fn assert_rent_exempt(rent: &Rent, account_info: &AccountInfo) -> ProgramResult {
    if !rent.is_exempt(account_info.lamports(), account_info.data_len()) {
        msg!(
            "Rent exempt balance insufficient got {} expected {}",
            &account_info.lamports().to_string(),
            &rent.minimum_balance(account_info.data_len()).to_string(),
        );
        Err(BridgeError::NotRentExempt.into())
    } else {
        Ok(())
    }
}

/// Issue a spl_token `Transfer` instruction.
#[inline(always)]
fn spl_token_transfer(params: TokenTransferParams<'_, '_>) -> ProgramResult {
    let TokenTransferParams {
        source,
        destination,
        authority,
        token_program,
        amount,
        authority_signer_seeds,
    } = params;
    let result = invoke_optionally_signed(
        &spl_token::instruction::transfer(
            token_program.key,
            source.key,
            destination.key,
            authority.key,
            &[],
            amount,
        )?,
        &[source, destination, authority, token_program],
        authority_signer_seeds,
    );
    result.map_err(|_| BridgeError::TokenTransferFailed.into())
}

/// Issue a spl_token `Close` instruction.
#[inline(always)]
fn spl_close_token_acc(params: TokenCloseParams<'_, '_>) -> ProgramResult {
    let TokenCloseParams {
        account,
        destination,
        authority,
        token_program,
        authority_signer_seeds,
    } = params;
    let result = invoke_optionally_signed(
        &spl_token::instruction::close_account(
            token_program.key,
            account.key,
            destination.key,
            authority.key,
            &[],
        )?,
        &[account, destination, authority, token_program],
        authority_signer_seeds,
    );
    result.map_err(|_| BridgeError::CloseTokenAccountFailed.into())
}

/// Invoke signed unless signers seeds are empty
#[inline(always)]
fn invoke_optionally_signed(
    instruction: &Instruction,
    account_infos: &[AccountInfo],
    authority_signer_seeds: &[&[u8]],
) -> ProgramResult {
    if authority_signer_seeds.is_empty() {
        invoke(instruction, account_infos)
    } else {
        invoke_signed(instruction, account_infos, &[authority_signer_seeds])
    }
}

struct TokenTransferParams<'a: 'b, 'b> {
    source: AccountInfo<'a>,
    destination: AccountInfo<'a>,
    amount: u64,
    authority: AccountInfo<'a>,
    authority_signer_seeds: &'b [&'b [u8]],
    token_program: AccountInfo<'a>,
}

struct TokenCloseParams<'a: 'b, 'b> {
    account: AccountInfo<'a>,
    destination: AccountInfo<'a>,
    authority: AccountInfo<'a>,
    authority_signer_seeds: &'b [&'b [u8]],
    token_program: AccountInfo<'a>,
}

fn assert_uninitialized<T: Pack + IsInitialized>(
    account_info: &AccountInfo,
) -> Result<T, ProgramError> {
    let account: T = T::unpack_unchecked(&account_info.data.borrow())?;
    if account.is_initialized() {
        Err(BridgeError::AccInitialized.into())
    } else {
        Ok(account)
    }
}

fn instruction_in_merkle_tree(
    leaf: &[u8; 32],
    root: &[u8; 32],
    paths: &Vec<[u8; 32]>,
    path_lefts: &Vec<bool>
) -> bool {
    if paths.len() != path_lefts.len() {
        msg!("paths and path_lefts is not match");
        return false;
    }
    let mut build_root = leaf.clone();
    for i in 0..paths.len() {
        if path_lefts[i] {
            let mut temp = paths[i][..].to_vec();
            temp.extend_from_slice(&build_root[..]);
            build_root = hash(&temp[..]).to_bytes();
        } else if paths[i] == [0; 32] {
            let mut temp = build_root[..].to_vec();
            temp.extend_from_slice(&build_root[..]);
            build_root = hash(&temp[..]).to_bytes();
        } else {
            let mut temp = build_root[..].to_vec();
            temp.extend_from_slice(&paths[i][..]);
            build_root = hash(&temp[..]).to_bytes();
        }
    }
    build_root == *root
}

fn append_at_top(input: u64) -> Vec<u8>  {
    let mut  input_vec = input.to_be_bytes().to_vec();
    for _ in 0..24 {
        input_vec.insert(0, 0);
    }

    input_vec
}
