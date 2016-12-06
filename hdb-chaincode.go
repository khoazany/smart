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
	"strconv"
	"strings"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"encoding/json"
	// "regexp"
	"time"
)

var logger = shim.NewLogger("HDBChaincode")
var activitiesStr = "_activities"
var activityCountStr = "_activityCount"

// ============================================================================================================================
// ACTOR TYPE
// ============================================================================================================================
const ADMIN = "admin"
const USER = "user"
const VENDOR = "vendor"
const BUSINESS = "business"

// ============================================================================================================================
// HANDLE TIME
// ============================================================================================================================
const (
	millisPerSecond     = int64(time.Second / time.Millisecond)
	nanosPerMillisecond = int64(time.Millisecond / time.Nanosecond)
)

func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(msInt/millisPerSecond,
		(msInt%millisPerSecond)*nanosPerMillisecond), nil
}

// ============================================================================================================================
// ACTIVITY
// ============================================================================================================================
type Activity struct {
	ActivityId int64 `json:"activityId"`
	Actor Actor `json:"actor"`
	ActivityType string `json:"activityType"`
	Kiosk Kiosk `json:"kiosk"`
	Resources Resources `json:"resources"`
	Device Device `json:"device"`
	Remark string `json:"remark"`
	Timestamp int64 `json:"timestamp"`			//utc timestamp of creation
}

type AllActivities struct {
	Activities []Activity `json:"activity"`
}

type Device struct {
	DeviceType string `json:"deviceType"`
	Id1 string `json:"id1"`
	Id2 string `json:"id2"`
	Id3 string `json:"id3"`
	Id4 string `json:"id4"`
}

type Actor struct {
	ActorType string `json:"actorType"`
	Name string `json:"name"`
	Telephone string `json:"telephone"`
	Email string `json:"email"`
}

type Kiosk struct {
	KioskId string `json:"kioskId"`
	Latitude float64 `json:"float64"`
	Longitude float64 `json:"float64"`
	Details string `json:"details"`
}

type Resources struct {
	Resources []Resource `json:"resources"`
}

type Resource struct {
	ResourceOwner string `json:"resourceOwner"`
	ResourceType string `json:"resourceType"`
	ResourceId string `json:"resourceId"`
	Details string `json:"details"`
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
	for i:=0; i < len(args); i=i+2 {
		t.add_ecert(stub, args[i], args[i+1])
	}

	return nil, nil
}

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	// caller, caller_affiliation, err := t.get_caller_data(stub)

	// if err != nil { fmt.Printf("CREATE_ACTIVITY: Error retrieving caller information: %s", err); return nil, errors.New("Error retrieving caller information")}
	var caller, caller_affiliation string

	logger.Debug("function: ", function)
    logger.Debug("caller: ", caller)
    logger.Debug("affiliation: ", caller_affiliation)

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
	if function == "view_activities" {											//read a variable
		return t.view_activities(stub, args)
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

func (t *SimpleChaincode) view_activities(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	// var key, jsonResp string
	var err error

	// if len(args) != 1 {
	// 	return nil, errors.New("Incorrect number of arguments. Expecting 1")
	// }

	// get the activities struct
	activitiesAsBytes, err := stub.GetState(activitiesStr)
	if err != nil { fmt.Printf("VIEW_ACTIVITIES: Failed to retrieve activities: %s", err); return nil, errors.New("Failed to retrieve activities") }
	
	var activities AllActivities
	json.Unmarshal(activitiesAsBytes, &activities)
	var returnActivities []Activity

	activityIds, err := sliceAtoi64(strings.Split(args[0], ","))
	if err != nil { fmt.Printf("VIEW_ACTIVITIES: Failed to retrieve activityIds argument: %s", err); return nil, errors.New("Failed to retrieve activityIds argument") }

	// actorTypes := args[1]
	// names := args[2]
	// telephones := args[3]
	// emails := args[4]
	// activityTypes := args[5]
	// kioskIds := args[6]
	// kioskDetails := args[7]
	// resourceOwners := args[8]
	// resourcesTypes := args[9]
	// cabinets := args[10]

	for i:= range activities.Activities {
		var activity = activities.Activities[i]

		if (len(activityIds) > 0 && !containsInt64(activityIds, activity.ActivityId)) {
			continue
		}

		returnActivities = append(returnActivities, activity)
	}

	returnActivitiesBytes, err := json.Marshal(returnActivities)
	if err != nil { fmt.Printf("VIEW_ACTIVITIES: Failed to convert activities: %s", err); return nil, errors.New("Failed to convert activities") }

	return returnActivitiesBytes, nil
}

func sliceAtoi64(sa []string) ([]int64, error) {
    si := make([]int64, 0, len(sa))
    for _, a := range sa {
        i, err := strconv.ParseInt(a,10,64)
        if err != nil {
            return si, err
        }
        si = append(si, i)
    }
    return si, nil
}

func containsInt64(slice []int64, item int64) bool {
    set := make(map[int64]struct{}, len(slice))
    for _, s := range slice {
        set[s] = struct{}{}
    }

    _, ok := set[item] 
    return ok
}

//=================================================================================================================================
//	 Create Function
//=================================================================================================================================
//	 Create Vehicle - Creates the initial JSON for the vehcile and then saves it to the ledger.
//=================================================================================================================================
func (t *SimpleChaincode) create_activity(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string, args []string) ([]byte, error) {

	// if 	caller_affiliation != ADMIN {							// Only the admin can create new activity
	// 	fmt.Printf("CREATE_ACTIVITY: Permission Denied"); return nil, errors.New("Permission Denied")
	// }

	activityCountAsBytes, err := stub.GetState(activityCountStr)
	if err != nil { fmt.Printf("CREATE_ACTIVITY: Error when retrieving activity count: %s", err); return nil, errors.New("Error when retrieving activity count") }

	var activityCount int64
	activityCount = int64(binary.LittleEndian.Uint64(activityCountAsBytes))

	activityId       := activityCount
	actor            := Actor{ActorType: args[0], Name: args[1], Telephone: args[2], Email: args[3]}
	activityType     := args[4]
	latitude, err := strconv.ParseFloat(args[6], 64)
	if err != nil { fmt.Printf("CREATE_ACTIVITY: Invalid latitude format: %s", err); return nil, errors.New("Invalid latitude format") }	
	longitude, err := strconv.ParseFloat(args[7], 64)
	if err != nil { fmt.Printf("CREATE_ACTIVITY: Invalid longitude format: %s", err); return nil, errors.New("Invalid longitude format") }

	kiosk            := Kiosk{KioskId: args[5], Latitude: latitude, Longitude: longitude, Details: args[8]}
	remark           := args[9]
	timestamp        := makeTimestamp()	
	device           := Device{DeviceType: args[10], Id1: args[11], Id2: args[12], Id3: args[13], Id4: args[14]}

	var resources Resources
	for i:=15;i < len(args);i=i+4 {
		resource := Resource{ResourceOwner: args[i], ResourceType: args[i+1], ResourceId: args[i+2], Details: args[i+3]}
		resources.Resources = append(resources.Resources, resource)
	}

	// activity_json := "{" + token + actor + activityType + kioskId + resourceId + resourceName + resourceType + remark + "}" 	// Concatenates the variables to create the total JSON object

	// err = json.Unmarshal([]byte(activity_json), &a)							// Convert the JSON defined above into an activity object for go

	// 																	if err != nil { return nil, errors.New("Invalid JSON object") }
	// _, err  = t.save_changes(stub, a)

	var activity = Activity{ActivityId: activityId, Actor: actor, ActivityType: activityType, Kiosk: kiosk, Resources: resources, Remark: remark, Timestamp: timestamp, Device: device}
	// activityBytes, err := json.Marshal(&activity)
	// if err != nil { fmt.Printf("CREATE_ACTIVITY: Error saving changes: %s", err); return nil, errors.New("Error saving changes") }

    // get the activities struct
	activitiesAsBytes, err := stub.GetState(activitiesStr)
	if err != nil { fmt.Printf("CREATE_ACTIVITY: Failed to retrieve activities: %s", err); return nil, errors.New("Failed to retrieve activities") }
	var activities AllActivities
	json.Unmarshal(activitiesAsBytes, &activities)

	activities.Activities = append(activities.Activities, activity)
	fmt.Println("CREATE_ACTIVITY: Add new activity")
	jsonAsBytes, err := json.Marshal(activities)
	if err != nil { fmt.Printf("CREATE_ACTIVITY: Failed to update activities: %s", err); return nil, errors.New("Failed to update activities") }
	
	err = stub.PutState(activitiesStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	activityCount = activityCount + 1
	err = stub.PutState(activityCountStr, activityCount)
	if err != nil {
		return nil, err
	}

	fmt.Println("CREATE_ACTIVITY: End create activity process")																	
	// bytes, err := stub.GetState("tokens")
	// 																	if err != nil { return nil, errors.New("Unable to get tokens") }

	// var tokens Tokens_Holder

	// err = json.Unmarshal(bytes, &tokens)
	// 																	if err != nil {	return nil, errors.New("Corrupt Token_Holder record") }

	// tokens.Tokens = append(tokens.Tokens, token)

	// bytes, err = json.Marshal(tokens)
	// 														if err != nil { fmt.Print("Error creating Token_Holder record") }

	// err = stub.PutState("tokens", bytes)

	// 														if err != nil { return nil, errors.New("Unable to put the state") }

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

    username, err := stub.ReadCertAttribute("account");
	if err != nil { return "", errors.New("Couldn't get attribute 'account'. Error: " + err.Error()) }
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

func (t *SimpleChaincode) get_caller_data(stub shim.ChaincodeStubInterface) (string, string, error) {

	user, err := t.get_username(stub)

	logger.Debug("user: ", user)

	affiliation, err := t.check_affiliation(stub);

    if err != nil { return "", "", err }

	return user, affiliation, nil
}

// ============================================================================================================================
// Make Timestamp - create a timestamp in ms
// ============================================================================================================================
func makeTimestamp() int64 {
    return time.Now().UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond))
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
