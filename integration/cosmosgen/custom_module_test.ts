import { describe, expect, it } from 'vitest'
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing'
import { isDeliverTxSuccess } from '@cosmjs/stargate'

describe('custom module', async () => {
  const { Entry } = await import('chain-disco-js')
  const { txClient, queryClient } = await import('chain-disco-js/module')

  it('should create a list entry', async () => {
    const { account1 } = global.accounts

    const mnemonic = account1['Mnemonic']
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic)
    const [account] = await wallet.getAccounts();
    const tx = await txClient(wallet, { addr: global.txApi })

    const entryData: Entry = {
      id: '0',
      creator: account.address,
      name: "test",
    }

    // Create a new list entry
    const result = await tx.signAndBroadcast([
      tx.msgCreateEntry(entryData),
    ])

    expect(isDeliverTxSuccess(result)).toEqual(true)

    // Check that the list entry is created
    const query = await queryClient({ addr: global.queryApi })
    const response = await query.queryEntryAll()

    expect(response.statusText).toEqual('OK')
    expect(response.data['Entry']).toHaveLength(1)
    expect(response.data['Entry'][0]).toEqual(entryData)
  })
})
