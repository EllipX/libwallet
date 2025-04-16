# libwallet API

## Info

### `Info:ping`

Ping the lib to check if everything is doing fine.

### `Info:version`

Returns the lib's version

### `Info:first_run`

Return the date/time of the first run based on the storage endpoint

### `Info:onboarding`

returns an object with the state of the user's onboarding, useful to check if we need to prompt the user to create or restore a wallet

## Crash

* `GET Crash` Lists all crash events sorted by creation time
* `GET Crash/<id>` Fetch details of a specific crash event
* `DELETE Crash/<id>` Delete a specific crash event

## Lifecycle

* `Lifecycle:update`

## StoreKey

* `StoreKey:new` *REMOVED* use `StoreKey:create`
* `StoreKey:create` Returns a store key and its public key in PKIX format (private/public)
* `StoreKey:derivePassword` returns the public key for a given password based on the password and WalletKeyId
  * Password
  * WalletKeyId

## RemoteKey

* `RemoteKey:new` takes: `number` (intl format), return `session`
* `RemoteKey:reshare` takes: `key`, return `session` to initialize a key reshare
* `RemoteKey:validate` takes: `session` (returned by new or reshare), `code`, returns `RemoteKey`

## Wallet

* `GET`
* `POST Wallet` to create a new Wallet
  * `Name`
  * `Keys`: [ {"Type": "StoreKey", "Key": storeKey}, {"Type": "RemoteKey", "Key": remoteKey}, {"Type": "Password", "Key": password} ]
* `PATCH Wallet/<id>`
  * `Name`
* `DELETE Wallet/<id>` delete a wallet, its accounts, and everything
* ~~`GET Wallet:backup` Generate backup of all local wallet data for icloud/etc~~ **Use Wallet:restore instead**
* `GET Wallet/<id>:backup` Generate backup of a given wallet for icloud/etc
* `POST Wallet:restore` Restore/refresh/sync data from icloud backup
  * `files` : [ {"filename": "xxx", "data": "yyy"}, {...}, ...]
  * The api will respond with the following:
    * `update` if true, the backup is too old and needs to be generated again (call Wallet:backup and upload the data)
    * `delete` is an optional array of string. If specified, the files listed here should be deleted from the backup (old or deprecated)
    * `errors` if specified, means the restore failed on any of the files. Errors will contain the filename and the details.
    * `backup` if specified, contains a array in the form `[{"filename":"...","data":"..."},...]` (the same as backup) which are files that should be written to the backup
    * `update_count` number of items updated from this restore operation
    * `existing_count` number of items that already existed and do not need to be updated
    * `missing_count` number of items missing from the backup
* `POST Wallet/<id>:reshare` Reshare wallet keys among a new set of key holders
  * `Old` Array of key descriptions to be replaced `[]*wltsign.KeyDescription`
  * `New` Array of new key descriptions `[]*wltsign.KeyDescription`

## Wallet/Key

* `GET Wallet/Key/<id>`
* `POST Wallet/Key/<id>:recrypt` allows changing the password of a wallet key
  * `"Old": {"Type": "Password", "Key": password}`
  * `"New": {"Type": "Password", "Key": newpassword}`

## Network

* `GET Network`
  * TestNet=false (optional): if set to false, exclude testnets
* `GET Network/<id>`
* `POST Network`
  * Type == evm
  * ChainId
  * Name
  * RPC (=auto)
  * CurrencySymbol
  * BlockExplorer (=auto)
  * TestNet (bool)
  * Priority (int, larger values returned first)
* `Network/<id>:setCurrent`
* `PATCH Network/id`
* `DELETE Network/id`
* `POST Network:testRPC`
  * `URL` URL of RPC server to test

## Account

* `GET Account`
  * `Wallet` to list only accounts linked to a specific wallet
* `GET Account/<id>`
* `POST Account`
  * `Name`
  * `Wallet` Id of attached wallet
  * `Type` ethereum or bitcoin
  * `Index` Index of the account (starts at zero, two accounts of the same wallet / type / index will have the same address)
* `PATCH Account/<id>`
  * `Name`
* `DELETE Account/<id>` Delete an account and everything related
* `Account/<id>:setCurrent`

## Asset

* `GET` (list only)
  * _convert=USD (add FiatAmount and FiatCurrency to each asset with converted amount, can accept USD/EUR/GBP/JPY)

## Transaction

* `GET Transaction`
  * From: limit transaction list to a given account
  * Network: find transactions on a given network
  * _convert=USD (add FiatAmount and FiatCurrency to each asset with converted amount, can accept USD/EUR/GBP/JPY)
* `GET Transaction/<id>`
* `Transaction:validate` Validates if a transaction is OK, returns errors if anything seems wrong
* `Transaction:signAndSend`
  * Same params as `Transaction:validate` plus:
  * Keys: [ {"Id": "wkey-xxx", "Key": privateKey, {"Id": "wkey-yyy", "Key": password} ]
* `DELETE Transaction`
  * From: limit transaction deletion to a given account
  * Network: delete transactions on a given network
  * If no parameter is passed, ALL of the transaction history will be cleared
* `DELETE Transaction/id`

## Contact

* `GET Contact`
* `GET Contact/id`
* `POST Contact`
  * Name
  * Address
  * Type
  * Memo
* `PATCH Contact/id`
* `DELETE Contact/id`

## Web3

* `POST Web3:request`
  * `url` URL making the web3 requrest
  * `query` Content of the query, an object with `method` and optionally `params`

## Web3/Connection

Web3/Connection manages which sites have access to which accounts

* `GET Web3/Connection`
* `GET Web3/Connection/<id>`
  * Host: list only connections for a given host
* `POST Web3/Connection`
  * Host: hostname of the connected site
  * Account: id of the connected account
* `DELETE Web3/Connection/<id>`

## Request

* EVENT: `{"result":"event","event":"request","data":{"request_id":"..."}}` A new request is PENDING
* `GET Request:test` to run a test on the event
* `GET Request/<id>` to fetch a given request including its details (request, etc)
  * Type can be one of: connect, sign, personal_sign, add_network, change_network, test
  * Status can be one of: pending, accepted, rejected, timedout
  * Transaction can be optionally included if request is for sign
  * Value can be optionally included, is context of the request (will replace Transaction)
* `POST Request/<id>:approve`
  * Must pass Accounts as an array of account IDs if the request Type is connect
* `POST Request/<id>:reject`

## Token (ERC20)

* `GET Token`
* `GET Token/id`
* `POST Contact`
  * Network => network Id
  * Type => ethereum || bitcoin
  * Address
  * Name
  * Symbol
  * Decimals
  * TotalSupply
* `DELETE Token/id`