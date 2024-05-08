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
    AccountChangeCallback,
    GetProgramAccountsFilter,
    MemcmpFilter,
} from "@solana/web3.js";

import {
    createInitializeMintInstruction,
    TOKEN_PROGRAM_ID,
} from "@solana/spl-token";

const accountKey = new PublicKey(
    "BKGhwbiTHdUxcuWzZtDWyioRBieDEXTtgEk8u1zskZnk"
);

const INCOGNIO_PROXY_KEY = "8WUP1RGTDTZGYBjkHQfjnwMbnnk25hnE6Du7vFpaq1QK";
const SHIELD_INST = 'Shield';
const UNSHIELD_INST = 'Unshield';

var shieldDict = {} as any;

// connection
const connection = new Connection(clusterApiUrl("devnet"), "confirmed");
(async () => {
    // var filter: GetProgramAccountsFilter = {
    //     memcmp: {
    //         offset: 32,
    //         bytes: "EwQDxcUNKYcJpXMwcfwxYV9QDy6hq6DdQ9DQF5yEFitT",
    //     }
    // };
    // connection.onProgramAccountChange(
    //     accountKey,
    //     makeShield,
    //     'finalized',
    //     [filter],
    // );

    connection.onLogs(
        accountKey,
        makeShield,
        'finalized'
    );
})();

console.log("cron job process unshield");

function makeShield(result: any) {
    if (result.err != null) {
        console.log("tx failed: ", result?.signature);
        return;
    }

    if (!result?.signature) {
        console.log("invalid signature in tx: ", result?.signature);
        return;
    }

    console.log("========= extract data shield from tx ==========");
    let logs = result?.logs;
    if (logs == null || logs.length < 6) {
        console.log("invalid log: ", logs);
        return;
    }

    let instruction = logs[1];
    if (!instruction || !instruction.includes(SHIELD_INST)) {
        console.log("not shield instruction ", instruction);
        return;
    }

    const shieldLog = logs[6];
    const shieldLogSplit = shieldLog.split(":");
    if (shieldLogSplit.length < 3) {
        console.log("invalid split please check ", shieldLogSplit);
        return;
    }

    const shieldInfos = shieldLogSplit[2].split(",");
    if (shieldInfos.length < 4) {
        console.log("invalid shield info split please check ", shieldInfos);
        return;
    }

    const incognitoProxy = shieldInfos[0];
    if (incognitoProxy != INCOGNIO_PROXY_KEY) {
        console.log(`invalid incognito proxy key expected ${INCOGNIO_PROXY_KEY} got ${incognitoProxy} `);
        return;
    }

    // todo: query to db to check
    if (shieldDict[result?.signature]) {
        console.log("signature in tx existed: ", result?.signature);
        return;
    }
    shieldDict[result?.signature] = true;

    const incAddress = shieldInfos[1];
    const token = shieldInfos[2];
    const amount = shieldInfos[3];

    console.log(`incognito proxy ${incognitoProxy} shield address ${incAddress} token ${token} amount ${amount}`);

    // todo: call incognito rpc to shield
}