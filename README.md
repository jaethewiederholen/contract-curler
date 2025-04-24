# Contract Curler

A CLI tool for interacting with Ethereum smart contract view functions via curl commands.

## Features

- Generate curl commands to call smart contract view functions
- Encode function calls with proper ABI encoding
- Parse and format return values based on the function's return type signature
- Interactive CLI interface
- Support for custom Ethereum RPC endpoints

## Installation

1. Make sure you have Go installed (version 1.23 or later)
2. Clone this repository
3. Run `go mod download` to download dependencies
4. Build with `go build -o contract-curler`

## Usage

1. Run the tool:
   ```
   ./contract-curler
   ```

2. Follow the interactive prompts:
   - Enter the contract address
   - Enter the function signature (e.g., `getTransaction(uint256)`)
   - Enter any required function arguments
   - Enter the function return type (e.g., `(address,uint256,bytes,bool,uint256)`)
   - **Enter the Ethereum RPC URL** (e.g., https://mainnet.infura.io/v3/YOUR_API_KEY)
     - If left blank, defaults to http://localhost:8545

3. The tool will generate a curl command that you can either:
   - Execute directly through the tool
   - Copy and run manually in your terminal

4. If executed, the tool will display both the raw response and a human-readable decoded version based on the return type you specified.

## Example

```
Enter contract address: 0x8409afBddbAF10d0C94F3ff330925F9F0a4b1F6c
Enter function signature (e.g., 'getBalance(address)'): getTransaction(uint256)
Enter value for parameter 1 (uint256): 3
Enter return type (e.g., '(uint256,address)'): (address,uint256,bytes,bool,uint256)
Enter Ethereum RPC URL (default: http://localhost:8545): https://mainnet.infura.io/v3/YOUR_API_KEY

Generated curl command:
curl -X POST https://mainnet.infura.io/v3/YOUR_API_KEY -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"0x8409afBddbAF10d0C94F3ff330925F9F0a4b1F6c","data":"0x4f0f4aa90000000000000000000000000000000000000000000000000000000000000003"},"latest"],"id":1}'

Do you want to execute this command? (y/n): y

Raw Response:
{"jsonrpc":"2.0","id":1,"result":"0x000000000000000000000000..."}

Decoded Result:
address: 0x1234...
uint256: 42
bytes: 0xabcd...
bool: true
uint256: 123456
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 