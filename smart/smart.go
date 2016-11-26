/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	// "strconv"
	// "strings"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"encoding/json"
	// "regexp"
)

var logger = shim.NewLogger("HDBChaincode")

// ============================================================================================================================
// ACTOR TYPE
// ============================================================================================================================
const ADMIN = "admin"
const USER = "user"
const VENDOR = "vendor"

// ============================================================================================================================
// ACTIVITY
// ============================================================================================================================
type Activity struct {
	Token string `json:"token"`
	Actor string `json:"actor"`
	ActivityType string `json:"activityType"`
	KioskId string `json:"kioskId"`
	ResourceType string `json:"resourceType"`
	ResourceId string `json:"resourceId"`
	ResourceName string `json:"resourceName"`
	Device Device `json:"device"`
	Remark string `json:"remark"`
}

type Device struct {
	DeviceType string `json:"deviceType"`
	Id1 string `json:"id1"`
	Id2 string `json:"id2"`
	Id3 string `json:"id3"`
	Id4 string `json:"id4"`
}

//==============================================================================================================================
//	Token Holder - Defines the structure that holds all the tokens for activities that have been created.
//				Used as an index when querying all tokens.
//==============================================================================================================================

type Tokens_Holder struct {
	Tokens 	[]string `json:"tokens"`
}

//==============================================================================================================================
//	Actor_and_eCert - Struct for storing the JSON of an actor and their ecert
//==============================================================================================================================
type Actor_and_eCert struct {
	Identity string `json:"identity"`
	eCert string `json:"ecert"`
}


// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

// Init resets all the things
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	return nil, nil
}

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	caller, caller_affiliation, err := t.get_caller_data(stub)

	if err != nil { return nil, errors.New("Error retrieving caller information")}

	// Handle different functions	
	if function == "create_activity" {													//initialize the chaincode state, used as reset
		return t.create_activity(stub, caller, caller_affiliation, args)
	} else if function == "write" {
		return t.write(stub, args)
	}

	fmt.Println("invoke did not find func: " + function)					//error

	return nil, errors.New("Received unknown function invocation: " + function)
}

func (t *SimpleChaincode) write(stub shim.ChaincodeStubInterface, args[] string) ([]byte, error) {
	var key, value string
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the key and value to set")
	}

	key = args[0]
	value = args[1]
	err = stub.PutState(key, []byte(value))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" {											//read a variable
		return t.read(stub, args)
	}
	fmt.Println("query did not find func: " + function)						//error

	return nil, errors.New("Received unknown function query: " + function)
}

func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	key = args[0]
	valAsbytes, err := stub.GetState(key)
	
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil
}

//=================================================================================================================================
//	 Create Function
//=================================================================================================================================
//	 Create Vehicle - Creates the initial JSON for the vehcile and then saves it to the ledger.
//=================================================================================================================================
func (t *SimpleChaincode) create_activity(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string, args []string) ([]byte, error) {
	
	if 	caller_affiliation != ADMIN && caller_affiliation != VENDOR {							// Only the admin can create a new v5c
		return nil, errors.New("Permission Denied. create_activity")
	}

	var a Activity

	token         := "\"Token\":\"" + args[0] + "\", "
	actor         := "\"Actor\":\"" + args[1] + "\", "
	activityType   := "\"ActivityType\":\"" + args[2] + "\", "
	kioskId   := "\"KioskId\":\"" + args[3] + "\", "
	resourceId   := "\"ResourceId\":\"" + args[4] + "\", "
	resourceName   := "\"ResourceName\":\"" + args[5] + "\", "
	resourceType   := "\"ResourceType\":\"" + args[6] + "\", "
	deviceJson, err := t.device_to_json(args[7],args[8],args[9],args[10],args[11])
																		if err != nil { return nil, errors.New("Error when reading device information") }

	device   := "\"Device\":" + deviceJson + ", "
	remark   := "\"Remark\":\"" + args[12] + "\""

	activity_json := "{" + token + actor + activityType + kioskId + resourceId + resourceName + resourceType + device + remark + "}" 	// Concatenates the variables to create the total JSON object

	err = json.Unmarshal([]byte(activity_json), &a)							// Convert the JSON defined above into an activity object for go

																		if err != nil { return nil, errors.New("Invalid JSON object") }
	_, err  = t.save_changes(stub, a)
																		if err != nil { fmt.Printf("CREATE_ACTIVITY: Error saving changes: %s", err); return nil, errors.New("Error saving changes") }

	bytes, err := stub.GetState("tokens")

																		if err != nil { return nil, errors.New("Unable to get tokens") }

	var tokens Tokens_Holder

	err = json.Unmarshal(bytes, &tokens)
																		if err != nil {	return nil, errors.New("Corrupt Token_Holder record") }

	tokens.Tokens = append(tokens.Tokens, token)

	bytes, err = json.Marshal(tokens)
															if err != nil { fmt.Print("Error creating Token_Holder record") }

	err = stub.PutState("tokens", bytes)

															if err != nil { return nil, errors.New("Unable to put the state") }

	return nil, nil
}

//==============================================================================================================================
//	 General Functions
//==============================================================================================================================
//	 get_ecert - Takes the name passed and calls out to the REST API for HyperLedger to retrieve the ecert
//				 for that user. Returns the ecert as retrived including html encoding.
//==============================================================================================================================
func (t *SimpleChaincode) get_ecert(stub shim.ChaincodeStubInterface, name string) ([]byte, error) {

	ecert, err := stub.GetState(name)

	if err != nil { return nil, errors.New("Couldn't retrieve ecert for user " + name) }

	return ecert, nil
}

//==============================================================================================================================
//	 add_ecert - Adds a new ecert and user pair to the table of ecerts
//==============================================================================================================================
func (t *SimpleChaincode) add_ecert(stub shim.ChaincodeStubInterface, name string, ecert string) ([]byte, error) {


	err := stub.PutState(name, []byte(ecert))

	if err == nil {
		return nil, errors.New("Error storing eCert for user " + name + " identity: " + ecert)
	}

	return nil, nil
}

//==============================================================================================================================
//	 get_caller - Retrieves the username of the user who invoked the chaincode.
//				  Returns the username as a string.
//==============================================================================================================================

func (t *SimpleChaincode) get_username(stub shim.ChaincodeStubInterface) (string, error) {

    username, err := stub.ReadCertAttribute("username");
	if err != nil { return "", errors.New("Couldn't get attribute 'username'. Error: " + err.Error()) }
	return string(username), nil
}

//==============================================================================================================================
//	 check_affiliation - Takes an ecert as a string, decodes it to remove html encoding then parses it and checks the
// 				  		certificates common name. The affiliation is stored as part of the common name.
//==============================================================================================================================

func (t *SimpleChaincode) check_affiliation(stub shim.ChaincodeStubInterface) (string, error) {
    affiliation, err := stub.ReadCertAttribute("role");
	if err != nil { return "", errors.New("Couldn't get attribute 'role'. Error: " + err.Error()) }
	return string(affiliation), nil

}

//==============================================================================================================================
//	 get_caller_data - Calls the get_ecert and check_role functions and returns the ecert and role for the
//					 name passed.
//==============================================================================================================================
func (t * SimpleChaincode) device_to_json(deviceType string, id1 string, id2 string, id3 string, id4 string) (string, error) {
	device := Device{DeviceType: deviceType, Id1: id1, Id2: id2, Id3: id3, Id4: id4}

	bytes, err := json.Marshal(device)

	if err != nil { fmt.Printf("SAVE_CHANGES: Error retrieving device json: %s", err); return "", errors.New("Error retrieving device json") }

	return string(bytes), nil
}

func (t *SimpleChaincode) get_caller_data(stub shim.ChaincodeStubInterface) (string, string, error) {

	user, err := t.get_username(stub)

	affiliation, err := t.check_affiliation(stub);

    if err != nil { return "", "", err }

	return user, affiliation, nil
}

//==============================================================================================================================
// save_changes - Writes to the ledger the Activity struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
//==============================================================================================================================
func (t *SimpleChaincode) save_changes(stub shim.ChaincodeStubInterface, a Activity) (bool, error) {

	bytes, err := json.Marshal(a)

	if err != nil { fmt.Printf("SAVE_CHANGES: Error converting activity record: %s", err); return false, errors.New("Error converting activity record") }

	err = stub.PutState(a.Token, bytes)

	if err != nil { fmt.Printf("SAVE_CHANGES: Error storing activity record: %s", err); return false, errors.New("Error storing activity record") }

	return true, nil
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
