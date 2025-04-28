package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"golang.org/x/crypto/sha3"
)

// JsonRpcRequest represents an Ethereum JSON-RPC request
type JsonRpcRequest struct {
	JsonRpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

// JsonRpcResponse represents an Ethereum JSON-RPC response
type JsonRpcResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  string `json:"result"`
}

// Function to encode method signature and parameters
func encodeMethodCall(methodSig string, args []string) (string, error) {
	// Extract function name and parameters
	re := regexp.MustCompile(`(\w+)\((.*)\)`)
	matches := re.FindStringSubmatch(methodSig)
	if len(matches) < 3 {
		return "", fmt.Errorf("invalid method signature format")
	}

	functionName := matches[1]
	paramTypesStr := matches[2]
	var paramTypes []string
	if paramTypesStr != "" {
		paramTypes = strings.Split(paramTypesStr, ",")
	}

	// Create function signature hash (first 4 bytes of keccak256 hash)
	methodSignature := functionName + "(" + strings.Join(paramTypes, ",") + ")"
	methodID := functionSelector(methodSignature)

	fmt.Println("Method ID:", methodID)

	// If no args, just return the method ID
	if len(paramTypes) == 0 || len(args) == 0 {
		return "0x" + methodID, nil
	}

	// Build ABI argument types
	var arguments abi.Arguments
	for _, paramType := range paramTypes {
		abiType, err := abi.NewType(strings.TrimSpace(paramType), "", nil)
		if err != nil {
			return "", fmt.Errorf("failed to parse ABI type '%s': %v", paramType, err)
		}
		arguments = append(arguments, abi.Argument{Type: abiType})
	}

	// Parse input arguments
	var values []interface{}
	for i, arg := range args {
		paramType := strings.TrimSpace(paramTypes[i])
		var value interface{}
		var err error

		switch {
		case strings.HasPrefix(paramType, "uint") || strings.HasPrefix(paramType, "int"):
			// Use big.Int for all integer types to handle uint256 properly
			bigInt := new(big.Int)
			_, success := bigInt.SetString(arg, 10)
			if !success {
				return "", fmt.Errorf("failed to parse integer argument '%s'", arg)
			}
			value = bigInt
		case paramType == "address":
			if !strings.HasPrefix(arg, "0x") {
				arg = "0x" + arg
			}
			value = common.HexToAddress(arg)
		case paramType == "bool":
			value, err = strconv.ParseBool(arg)
			if err != nil {
				return "", fmt.Errorf("failed to parse boolean argument: %v", err)
			}
		case strings.HasPrefix(paramType, "bytes"):
			if !strings.HasPrefix(arg, "0x") {
				arg = "0x" + arg
			}
			bytes, err := hexutil.Decode(arg)
			if err != nil {
				return "", fmt.Errorf("failed to decode bytes argument: %v", err)
			}
			value = bytes
		case paramType == "string":
			value = arg
		default:
			return "", fmt.Errorf("unsupported parameter type: %s", paramType)
		}

		values = append(values, value)
	}

	// Pack the arguments
	encodedArgs, err := arguments.Pack(values...)
	if err != nil {
		return "", fmt.Errorf("failed to encode arguments: %v", err)
	}

	// Combine method ID and encoded arguments
	return "0x" + methodID + hex.EncodeToString(encodedArgs), nil
}

// Function to decode return values
func decodeReturnValues(returnData string, returnTypes string) ([]interface{}, error) {
	// Parse return types
	returnTypesStr := strings.Trim(returnTypes, "()")
	var returnTypeList []string
	if returnTypesStr != "" {
		returnTypeList = strings.Split(returnTypesStr, ",")
	}

	// Remove 0x prefix if present
	if strings.HasPrefix(returnData, "0x") {
		returnData = returnData[2:]
	}

	// Decode hex data
	data, err := hex.DecodeString(returnData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode return data: %v", err)
	}

	// Build ABI return types
	var arguments abi.Arguments
	for _, typStr := range returnTypeList {
		typStr = strings.TrimSpace(typStr)
		abiType, err := abi.NewType(typStr, "", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse return type '%s': %v", typStr, err)
		}
		arguments = append(arguments, abi.Argument{Type: abiType})
	}

	// Unpack the return data
	values, err := arguments.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode return values: %v", err)
	}

	return values, nil
}

// Function to format return values for display
func formatReturnValues(values []interface{}, returnTypes []string) []string {
	results := make([]string, len(values))

	for i, val := range values {
		returnType := strings.TrimSpace(returnTypes[i])

		switch v := val.(type) {
		case common.Address:
			results[i] = fmt.Sprintf("%s: %s", returnType, v.Hex())
		case []byte:
			results[i] = fmt.Sprintf("%s: %s", returnType, hex.EncodeToString(v))
		case string:
			results[i] = fmt.Sprintf("%s: %s", returnType, v)
		case *big.Int:
			results[i] = fmt.Sprintf("%s: %s", returnType, v.String())
		default:
			results[i] = fmt.Sprintf("%s: %v", returnType, v)
		}
	}

	return results
}

func functionSelector(signature string) string {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(signature))
	hash := hasher.Sum(nil)
	selector := hash[:4]
	return fmt.Sprintf("%x", selector)
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	// Get contract address
	fmt.Print("Enter contract address: ")
	scanner.Scan()
	contractAddress := scanner.Text()

	// Get function signature
	fmt.Print("Enter function signature (e.g., getBalance(address)): ")
	scanner.Scan()
	functionSig := scanner.Text()

	// Extract function parameters from signature
	re := regexp.MustCompile(`\((.*)\)`)
	matches := re.FindStringSubmatch(functionSig)
	var paramTypes []string
	if len(matches) > 1 && matches[1] != "" {
		paramTypes = strings.Split(matches[1], ",")
	}

	// Get return type
	fmt.Print("Enter return type (e.g., (uint256,address)): ")
	scanner.Scan()
	returnType := scanner.Text()

	// Get arguments
	var args []string
	for i, paramType := range paramTypes {
		fmt.Printf("Enter value for parameter %d (%s): ", i+1, paramType)
		scanner.Scan()
		args = append(args, scanner.Text())
	}

	// Get RPC URL
	fmt.Print("Enter Ethereum RPC URL (default: http://localhost:8545): ")
	scanner.Scan()
	rpcURL := scanner.Text()
	if rpcURL == "" {
		rpcURL = "http://localhost:8545"
	}

	// Encode function call
	encodedData, err := encodeMethodCall(functionSig, args)
	if err != nil {
		fmt.Printf("Error encoding function call: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Encoded data:", encodedData)

	// Create JSON-RPC request
	request := JsonRpcRequest{
		JsonRpc: "2.0",
		Method:  "eth_call",
		Params: []interface{}{
			map[string]interface{}{
				"to":   contractAddress,
				"data": encodedData,
			},
			"latest",
		},
		Id: 1,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error creating JSON request: %v\n", err)
		os.Exit(1)
	}

	// Display the curl command
	curlCmd := fmt.Sprintf("curl -X POST %s -H \"Content-Type: application/json\" --data '%s'",
		rpcURL, string(jsonData))
	fmt.Println("\nGenerated curl command:")
	fmt.Println(curlCmd)

	// Ask if user wants to execute the command
	fmt.Print("\nDo you want to execute this command? (y/n): ")
	scanner.Scan()
	execute := scanner.Text()

	if strings.ToLower(execute) == "y" || strings.ToLower(execute) == "yes" {
		// Execute the request
		resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Error executing request: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		// Parse the response
		var response JsonRpcResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nRaw Response:")
		fmt.Println(string(body))

		// Parse the return types
		returnTypeStr := strings.Trim(returnType, "()")
		var returnTypeList []string
		if returnTypeStr != "" {
			returnTypeList = strings.Split(returnTypeStr, ",")
		}

		// Decode and display the result
		if response.Result != "" {
			fmt.Println("\nDecoded Result:")
			values, err := decodeReturnValues(response.Result, returnType)
			if err != nil {
				fmt.Printf("Error decoding results: %v\n", err)
				os.Exit(1)
			}

			formattedValues := formatReturnValues(values, returnTypeList)
			for _, value := range formattedValues {
				fmt.Println(value)
			}
		}
	}
}
