import {
    clusterApiUrl,
    Connection,
    Keypair,
    Transaction,
    SystemProgram,
    PublicKey,
    sendAndConfirmTransaction,
    TransactionInstruction,
    SYSVAR_RENT_PUBKEY,
} from "@solana/web3.js";
import {
    createInitializeMintInstruction,
    TOKEN_PROGRAM_ID,
    MINT_SIZE,
    getMinimumBalanceForRentExemptMint,
    createMint,
    createAssociatedTokenAccount,
    getAssociatedTokenAddress,
    createMintToCheckedInstruction,
    createAssociatedTokenAccountInstruction,
    mintToChecked,
    getAccount,
} from "@solana/spl-token";
import * as bs58 from "bs58";
import { deserialize, serialize } from "borsh";
import BN = require("bn.js");
// vault program
const programId = new PublicKey(
    "GqyrpvvqkcrnxF9TVhfz4CwXFTZ7QsXvjUJ5YaKTxsTH"
);

// connection
const connection = new Connection(clusterApiUrl("devnet"), "confirmed");

// 5YNmS1R9nNSCDzb5a7mMJ1dwK9uHeAAF4CmPEwKgVWr8
const feePayer = Keypair.fromSecretKey(
    bs58.decode("588FU4PktJWfGfxtzpAAXywSNt74AvtroVzGfKkVN1LwRuvHwKGr851uH8czM5qm4iqLbs1kKoMKtMJG4ATR7Ld2")
);
console.log(feePayer.publicKey.toBase58());

// G2FAbFQPFa5qKXCetoFZQEvF9BVvCKbvUZvodpVidnoY
const shieldMaker = Keypair.fromSecretKey(
    Uint8Array.from(Uint8Array.from([56,52,143,70,102,247,217,158,213,127,195,28,52,49,229,216,186,136,63,94,185,108,216,64,35,120,204,184,221,151,114,120,92,163,172,46,113,242,87,204,236,137,51,132,55,203,117,88,87,243,21,194,162,119,17,200,227,147,2,222,181,12,77,224]))
);

const swapMinter = Keypair.fromSecretKey(
    Uint8Array.from(Uint8Array.from([126,218,91,206,170,129,224,44,74,68,87,211,125,45,7,198,94,231,152,88,34,136,164,192,128,193,34,233,81,123,183,57,46,183,53,172,18,161,72,81,251,126,133,235,247,180,254,81,87,40,41,88,141,85,112,158,238,230,161,11,250,198,179,133]))
);

console.log(swapMinter.publicKey.toBase58());
console.log(toHexString(swapMinter.secretKey));

(async () => {

    // let mintPubkey = await createMint(
    //     connection, // conneciton
    //     feePayer, // fee payer
    //     alice.publicKey, // mint authority
    //     alice.publicKey, // freeze authority (you can use `null` to disable it. when you disable it, you can't turn it on again)
    //     8 // decimals
    // );
    // console.log(`mint: ${mintPubkey.toBase58()}`);
    const mintPubkey = new PublicKey("EHheP6Wfyz65ve258TYQcfBHAAY4LsErnmXZozrgfvGr");

    // {
    //     let ata = await createAssociatedTokenAccount(
    //         connection, // connection
    //         feePayer, // fee payer
    //         mintPubkey, // mint
    //         alice.publicKey // owner,
    //     );
    //     console.log(`ATA: ${ata.toBase58()}`);
    // }

    const shieldMakerAccount = new PublicKey("5397KrEBCuEhdTjWF5B9xjVzGJR6MyxXLP3srbrWo2gD");
    console.log("shield maker ", shieldMakerAccount.toBytes());
    console.log("token id ", mintPubkey.toBytes());
    const shield_amount = 1e12;
    // mint token to shield maker token account
    {
        let txhash = await mintToChecked(
            connection, // connection
            feePayer, // fee payer
            mintPubkey, // mint
            shieldMakerAccount, // receiver (sholud be a token account)
            shieldMaker, // mint authority
            1e12, // amount. if your decimals is 8, you mint 10^8 for 1 token.
            8 // decimals
        );
        console.log(`txhash: ${txhash}`);
    }

    //     // if your wallet is off-curve, you should use
    //     // let ata = await getAssociatedTokenAddress(
    //     //   mintPubkey, // mint
    //     //   alice.publicKey // owner
    //     //   true, // allowOwnerOffCurve
    //     // );
    //
    //     let tx = new Transaction().add(
    //         createAssociatedTokenAccountInstruction(
    //             feePayer.publicKey, // payer
    //             ata, // ata
    //             alice.publicKey, // owner
    //             mintPubkey // mint
    //         )
    //     );
    // const incognitoProxy = Keypair.generate();
    const incognitoProxy = Keypair.fromSecretKey(
        Uint8Array.from(Uint8Array.from([1,32,1,180,196,196,194,138,131,140,129,93,57,212,62,16,189,184,95,10,130,23,46,62,235,244,36,129,118,128,90,165,145,13,201,90,31,129,31,66,128,226,131,38,62,26,247,105,53,191,73,129,149,6,15,181,230,132,82,2,48,139,210,10]))
    );
    console.log(`incognito proxy: ${incognitoProxy.publicKey.toBase58()}`);
    console.log("incognito proxy private key: ", incognitoProxy.secretKey.toString());

    const beaconLengthInit = 1315 - 32;
    const lamportsExemptBeacon = await connection.getMinimumBalanceForRentExemption(beaconLengthInit, 'confirmed');

    const creatIncognitoInst = SystemProgram.createAccount({
        fromPubkey: feePayer.publicKey,
        newAccountPubkey: incognitoProxy.publicKey,
        lamports: lamportsExemptBeacon,
        space: beaconLengthInit,
        programId,
    });

    const [
        vaultTokenAuthority,
        bumpInit, // todo => store in incognito proxy
    ] = await PublicKey.findProgramAddress(
        [
            incognitoProxy.publicKey.toBuffer(),
        ],
        programId,
    );
    console.log("bump seed ", bumpInit);
    console.log("vault token authority ", vaultTokenAuthority.toBase58());
    let vaultTokenAcc = await getAssociatedTokenAddress(
        mintPubkey, // mint
        vaultTokenAuthority, // owner
      true, // allowOwnerOffCurve
    );
    console.log(" vault token account :", vaultTokenAcc.toBase58());
    // let tx = new Transaction().add(
    //     createAssociatedTokenAccountInstruction(
    //         feePayer.publicKey, // payer
    //         vaultTokenAcc, // ata
    //         vaultTokenAuthority, // owner
    //         mintPubkey // mint
    //     )
    // );

    console.log("=============== Init Beacon =================");

    // init beacon list address
    let beaconLength = 4;
    let beacon1 = [64,206,253,84,56,206,63,162,157,152,148,80,198,23,66,245,43,1,207,238,9,144,161,139,131,44,146,136,74,242,22,220,187,130,145,153,93,114,117,199,108,190,233,244,53,240,247,48,207,19,94,245,14,171,207,124,157,177,173,139,253,237,36,168];
    let beacon2 = [175,109,126,18,52,108,137,78,38,252,216,214,224,214,44,187,2,67,70,204,196,78,155,224,72,126,124,128,134,165,210,158,138,93,62,90,76,225,186,39,215,204,170,10,127,99,86,220,107,251,34,58,235,236,69,189,235,226,57,208,106,210,28,22];
    let beacon3 = [122,69,179,100,37,117,17,36,0,4,211,125,150,102,106,180,218,127,238,200,104,84,250,183,23,31,209,229,22,117,248,73,56,120,112,2,188,187,152,44,70,228,25,160,250,255,40,216,180,239,183,235,175,79,66,41,119,82,195,70,103,102,135,73];
    let beacon4 = [24,171,11,173,118,80,213,52,20,186,77,213,182,249,188,70,15,37,228,129,102,45,183,139,139,174,147,32,130,179,168,171,36,79,30,237,44,11,200,229,108,224,117,224,206,11,62,235,127,101,194,116,209,213,122,41,77,229,19,60,199,168,81,25];

    const init_beacon_instruction = new TransactionInstruction({
        keys: [
            {pubkey: SYSVAR_RENT_PUBKEY, isSigner: false, isWritable: false},
            {pubkey: incognitoProxy.publicKey, isSigner: false, isWritable: true},
        ],
        programId,
        data: Buffer.from(
            Uint8Array.of(
                2,
                ...new BN(bumpInit).toArray("le", 1),
                ...new BN(beaconLength).toArray("le", 1),
                ...beacon1,
                ...beacon2,
                ...beacon3,
                ...beacon4,
            )
        ),
    });
    console.log("Beacon instruction length: ", init_beacon_instruction.data.length);
    // todo: query beacon state before init

    const [
        vault_pda_account,
        _,
    ] = await PublicKey.findProgramAddress(
        [
            incognitoProxy.publicKey.toBuffer(),
            shieldMaker.publicKey.toBuffer(),
        ],
        programId,
    );

    let pubkeyArray:number[] = Array.from(shieldMaker.publicKey.toBytes());
    const init_vault_instruction = new TransactionInstruction({
        keys: [
            {pubkey: feePayer.publicKey, isSigner: true, isWritable: false},
            {pubkey: incognitoProxy.publicKey, isSigner: false, isWritable: false},
            {pubkey: vault_pda_account, isSigner: false, isWritable: true},
            {pubkey: SystemProgram.programId, isSigner: false, isWritable: false},
        ],
        programId,
        data: Buffer.from(
            Uint8Array.of(
                5,
                ...pubkeyArray,
            )
        ),
    });

    // create transaction init beacon list
    const trans_init_beacon = await setPayerAndBlockhashTransaction(
        [creatIncognitoInst, init_beacon_instruction, init_vault_instruction]
    );
    const signature_init_beacon = await signAndSendTransaction(trans_init_beacon, [feePayer, incognitoProxy]);
    await connection.confirmTransaction(signature_init_beacon);
    console.log(`init beacon txhash: ${signature_init_beacon}`);

    console.log("=============== Make shield request =================");

    let tokenAccount = await getAccount(connection, vaultTokenAcc);
    console.log(tokenAccount);
    // let incAddress = "";;
    var myBuffer:number[] = Array.from("12scKiKkL2ohYz6WF9zXGohgVqrJoRMtsbsJ8xhGNn1KNGhaEuW3SJEdPPTrhFxDJeG5wiyGr1BetJnok9Edrp4PhKxKAjF46UKTVAUTBMvD12ThrCqoDkr6WS7zSFoM9FvzP4xd6chZAtqfaTeq", (x) => x.charCodeAt(0));
    console.log("my buffer length ", myBuffer.length);
    const instruction = new TransactionInstruction({
        keys: [
            {pubkey: shieldMakerAccount, isSigner: false, isWritable: true},
            {pubkey: vaultTokenAcc, isSigner: false, isWritable: true},
            {pubkey: incognitoProxy.publicKey, isSigner: false, isWritable: false},
            {pubkey: shieldMaker.publicKey, isSigner: true, isWritable: false},
            {pubkey: TOKEN_PROGRAM_ID, isSigner: false, isWritable: false},
        ],
        programId,
        data: Buffer.from(
            Uint8Array.of(0, ...new BN(shield_amount).toArray("le", 8), ...myBuffer)
        ),
    });

    const trans = await setPayerAndBlockhashTransaction(
        [instruction]
    );
    const signature = await signAndSendTransaction(trans, [feePayer, shieldMaker]);
    const result = await connection.confirmTransaction(signature);
    console.log("end sendMessage", result);

    console.log("=============== Make unshield request =================");
    // create data
    let inst = [157,1,197,111,46,11,236,137,61,81,37,248,63,18,210,220,135,170,35,25,197,242,222,184,44,97,54,230,121,239,31,201,122,75,59,251,20,8,169,41,14,126,217,202,208,133,245,194,62,249,144,33,114,189,69,8,203,236,33,202,139,252,22,182,84,166,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,3,232,4,3,247,194,148,206,163,217,231,230,232,229,27,1,180,194,106,184,162,71,251,127,163,218,231,218,26,131,44,118,44,46,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];
    let height = 1;
    // =====
    let inst_paths = [[230,184,120,45,145,71,239,155,84,73,101,255,16,237,227,249,93,245,68,65,52,188,45,171,214,48,218,58,142,126,159,241],
        [50,19,205,111,207,123,115,108,252,197,54,235,131,223,254,252,171,24,167,244,113,233,90,51,224,58,124,77,61,171,131,135],
        [210,186,109,37,156,189,54,252,198,35,128,107,1,64,214,65,247,222,71,39,45,204,183,120,231,117,31,173,134,29,208,54],
        [50,36,225,183,195,238,157,68,33,143,0,255,168,100,62,38,48,211,141,19,176,16,50,39,182,180,229,59,184,184,220,14]
    ];
    // =====
    let inst_path_is_lefts = [true, true, true, false]
    // =====
    let inst_root = [7,169,244,251,166,215,250,217,140,251,162,81,245,74,209,160,7,248,255,76,108,60,61,23,109,24,25,83,168,226,21,24];
    let blkdata = [226,20,215,231,248,45,212,100,71,96,146,232,198,6,192,176,112,222,168,4,216,73,176,56,59,166,76,79,247,48,196,231];
    // ====
    let index = [0, 1, 2, 3]
    let signatures = [
        [3,187,120,11,108,18,253,108,227,230,220,165,168,161,190,175,135,80,191,45,10,66,167,84,231,6,184,94,157,137,69,145,57,43,58,134,34,59,33,25,101,171,231,215,234,194,76,86,67,200,202,238,9,109,165,119,184,2,49,145,190,198,54,79,1],
        [40,70,244,121,37,135,158,8,152,60,255,245,170,178,29,10,207,62,0,96,68,24,106,112,72,78,92,0,86,243,84,74,117,217,128,191,13,24,223,179,145,193,171,126,241,124,8,231,228,107,227,241,49,119,129,150,81,178,102,70,105,98,124,176,1],
        [144,13,147,72,12,116,178,7,244,251,37,209,229,191,63,5,33,176,38,227,104,22,102,47,65,139,251,136,180,54,93,60,28,92,127,207,228,68,255,171,186,168,130,186,12,22,134,216,43,233,158,236,91,185,13,44,50,173,83,15,157,162,57,202,1],
        [124,192,49,109,111,205,143,228,197,102,219,212,26,192,162,89,148,91,92,23,78,120,2,245,102,29,69,95,77,183,242,214,80,161,185,83,28,36,167,153,172,71,26,114,165,170,224,65,36,252,76,206,202,203,103,213,114,37,43,100,48,209,158,84,1]
    ];
    // =====
    const instruction_unshield = new TransactionInstruction({
        keys: [
            {pubkey: vaultTokenAcc, isSigner: false, isWritable: true},
            {pubkey: shieldMakerAccount, isSigner: false, isWritable: true},
            {pubkey: vaultTokenAuthority, isSigner: false, isWritable: false},
            {pubkey: vault_pda_account, isSigner: false, isWritable: true},
            {pubkey: incognitoProxy.publicKey, isSigner: false, isWritable: false},
            {pubkey: TOKEN_PROGRAM_ID, isSigner: false, isWritable: false},
        ],
        programId,
        data: Buffer.from(
            Uint8Array.of(
                1,
                ...inst,
                ...new BN(height).toArray("le", 8),
                ...new BN(inst_paths.length).toArray("le", 1),
                ...inst_paths[0],
                ...inst_paths[1],
                ...inst_paths[2],
                ...inst_paths[3],
                ...new BN(inst_path_is_lefts.length).toArray("le", 1),
                ...new BN(inst_path_is_lefts[0] ? 1 : 0).toArray("le", 1),
                ...new BN(inst_path_is_lefts[1] ? 1 : 0).toArray("le", 1),
                ...new BN(inst_path_is_lefts[2] ? 1 : 0).toArray("le", 1),
                ...new BN(inst_path_is_lefts[3] ? 1 : 0).toArray("le", 1),
                ...inst_root,
                ...blkdata,
                ...new BN(index.length).toArray("le", 1),
                ...new BN(index[0]).toArray("le", 1),
                ...new BN(index[1]).toArray("le", 1),
                ...new BN(index[2]).toArray("le", 1),
                ...new BN(index[3]).toArray("le", 1),
                ...new BN(signatures.length).toArray("le", 1),
                ...signatures[0],
                ...signatures[1],
                ...signatures[2],
                ...signatures[3],
            )
        ),
    });

    const unshield_trans = await setPayerAndBlockhashTransaction(
        [instruction_unshield]
    );
    const unshield_signature = await signAndSendTransaction(unshield_trans, [feePayer]);
    const unshield_result = await connection.confirmTransaction(unshield_signature);
    console.log("end sendMessage ", unshield_result);

})();


export async function setPayerAndBlockhashTransaction(instructions: any) {
    const transaction = new Transaction();
    instructions.forEach((element: any) => {
        transaction.add(element);
    });
    transaction.feePayer = feePayer.publicKey;
    let hash = await connection.getLatestBlockhash();
    transaction.recentBlockhash = hash.blockhash;
    return transaction;
}

async function signAndSendTransaction(transaction: any, listSigners : any) {
    try {
        console.log("start signAndSendTransaction");
        let txResult = await sendAndConfirmTransaction(
            connection,
            transaction,
            listSigners,
        );
        console.log("end signAndSendTransaction");
        return txResult;
    } catch (err) {
        console.log("signAndSendTransaction error", err);
        throw err;
    }
}

class ShieldDetails {
    constructor(properties: any) {
        Object.keys(properties).forEach((key) => {
            console.log({key});
            // this[key] = properties[key] as any;
        });
    }
    static schema = new Map([[ShieldDetails,
        {
            kind: 'struct',
            fields: [
                ['amount', 'u64'],
                ['inc_address', [148]]]
        }]]);
}

function toHexString(byteArray: any) {
    return Array.prototype.map.call(byteArray, function(byte) {
        return ('0' + (byte & 0xFF).toString(16)).slice(-2);
    }).join('');
}
