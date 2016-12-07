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
	// "strings"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"encoding/json"
	// "regexp"
	"time"
	// "encoding/binary"
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

// ============================================================================================================================
// ACTIVITY
// ============================================================================================================================
type Activity struct {
	ActivityId int64 `json:"activityId"`
	Actor Actor `json:"actor"`
	ActivityType string `json:"activityType"`
	Kiosk Kiosk `json:"kiosk"`
	Resources []Resource `json:"resources"`
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
	Latitude float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Details string `json:"details"`
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

	// Set initial activityCount = 0
	err := stub.PutState(activityCountStr, []byte(strconv.FormatInt(0,10)))
	if err != nil {
		return nil, err
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

	var activityIds []int64
	json.Unmarshal([]byte(args[0]), &activityIds)
	if err != nil { fmt.Printf("VIEW_ACTIVITIES: Failed to retrieve activityIds argument: %s", err); return nil, errors.New("Failed to retrieve activityIds argument") }

	var actorTypes []string
	json.Unmarshal([]byte(args[1]), &actorTypes)
	
	var names []string
	json.Unmarshal([]byte(args[2]), &names)

	var telephones []string
	json.Unmarshal([]byte(args[3]), &telephones)

	var emails []string
	json.Unmarshal([]byte(args[4]), &emails)

	var activityTypes []string
	json.Unmarshal([]byte(args[5]), &activityTypes)

	var kioskIds []string
	json.Unmarshal([]byte(args[6]), &kioskIds)

	var deviceTypes []string
	json.Unmarshal([]byte(args[7]), &deviceTypes)

	var id1s []string
	json.Unmarshal([]byte(args[8]), &id1s)

	var id2s []string
	json.Unmarshal([]byte(args[9]), &id2s)

	var id3s []string
	json.Unmarshal([]byte(args[10]), &id3s)

	var id4s []string
	json.Unmarshal([]byte(args[11]), &id4s)

	var resourceOwners []string
	json.Unmarshal([]byte(args[12]), &resourceOwners)

	var resourceTypes []string
	json.Unmarshal([]byte(args[13]), &resourceTypes)

	var resourceIds []string
	json.Unmarshal([]byte(args[14]), &resourceIds)

 	start := time.Time{}
	if (args[15] != "") {
		start, err = time.Parse("2006-01-02T15:04:05-0700", args[15])
		if err != nil { fmt.Printf("VIEW_ACTIVITIES: Invalid start time format: %s", err); return nil, errors.New("Invalid start time format") }
	}

	end := time.Time{}
	if (args[16] != "") {
	    end, err = time.Parse("2006-01-02T15:04:05-0700", args[16])
	    if err != nil { fmt.Printf("VIEW_ACTIVITIES: Invalid end time format: %s", err); return nil, errors.New("Invalid end time format") }
	}

	for i := range activities.Activities {
		var activity = activities.Activities[i]

		if (len(activityIds) > 0 && !containsInt64(activityIds, activity.ActivityId)) {
			continue
		}

		if (len(actorTypes) > 0 && !containsString(actorTypes, activity.Actor.ActorType)) {
			continue
		}

		if (len(names) > 0 && !containsString(names, activity.Actor.Name)) {
			continue
		}

		if (len(telephones) > 0 && !containsString(telephones, activity.Actor.Telephone)) {
			continue
		}

		if (len(emails) > 0 && !containsString(emails, activity.Actor.Email)) {
			continue
		}

		if (len(activityTypes) > 0 && !containsString(activityTypes, activity.ActivityType)) {
			continue
		}

		if (len(kioskIds) > 0 && !containsString(kioskIds, activity.Kiosk.KioskId)) {
			continue
		}

		if (len(resourceOwners) > 0 || len(resourceTypes) > 0 || len(resourceIds) > 0) {
			var existed = false
			for j := range activity.Resources {
				if (len(resourceOwners) > 0 && !containsString(resourceOwners, activity.Resources[j].ResourceOwner)) {
					continue
				}

				if (len(resourceTypes) > 0 && !containsString(resourceTypes, activity.Resources[j].ResourceType)) {
					continue
				}

				if (len(resourceIds) > 0 && !containsString(resourceIds, activity.Resources[j].ResourceId)) {
					continue
				}

				existed = true
			}

			if (!existed) {
				continue
			}
		}

		if (len(deviceTypes) > 0 && !containsString(deviceTypes, activity.Device.DeviceType)) {
			continue
		}

		if (len(id1s) > 0 && !containsString(id1s, activity.Device.Id1)) {
			continue
		}

		if (len(id2s) > 0 && !containsString(id2s, activity.Device.Id2)) {
			continue
		}

		if (len(id3s) > 0 && !containsString(id3s, activity.Device.Id3)) {
			continue
		}

		if (len(id4s) > 0 && !containsString(id4s, activity.Device.Id4)) {
			continue
		}

		if ((!start.IsZero() || !end.IsZero()) && !inTimeSpan(start,end,int64ToTime(activity.Timestamp))) {
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

func containsString(slice []string, item string) bool {
    set := make(map[string]struct{}, len(slice))
    for _, s := range slice {
        set[s] = struct{}{}
    }

    _, ok := set[item] 
    return ok
}

func inTimeSpan(start, end, check time.Time) bool {
	logger.Debug("start: ", start)
	logger.Debug("end: ", end)
	logger.Debug("check: ", check)

    return (start.IsZero() || check.After(start)) && (end.IsZero() || check.Before(end))
}

func int64ToTime(msInt int64) (time.Time) {

	return time.Unix(msInt/millisPerSecond,
		(msInt%millisPerSecond)*nanosPerMillisecond)
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

	activityCount, err := strconv.ParseInt(string(activityCountAsBytes), 10, 64)
	if err != nil { fmt.Printf("CREATE_ACTIVITY: Error when converting activity count: %s", err); return nil, errors.New("Error when converting activity count") }

	logger.Debug("activityCount: ", activityCount)

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

	logger.Debug("args: ", 14)

	var resources []Resource
	for i:=15;i < len(args);i=i+4 {
		resource := Resource{ResourceOwner: args[i], ResourceType: args[i+1], ResourceId: args[i+2], Details: args[i+3]}
		resources = append(resources, resource)
		logger.Debug("args: ", i+3)
	}

	// activity_json := "{" + token + actor + activityType + kioskId + resourceId + resourceName + resourceType + remark + "}" 	// Concatenates the variables to create the total JSON object

	// err = json.Unmarshal([]byte(activity_json), &a)							// Convert the JSON defined above into an activity object for go

	// 																	if err != nil { return nil, errors.New("Invalid JSON object") }
	// _, err  = t.save_changes(stub, a)

	activity := Activity{ActivityId: activityId, Actor: actor, ActivityType: activityType, Kiosk: kiosk, Resources: resources, Remark: remark, Timestamp: timestamp, Device: device}
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
	err = stub.PutState(activityCountStr, []byte(strconv.FormatInt(activityCount,10)))
	if err != nil {
		return nil, err
	}

	jsonAsBytes, err = json.Marshal(activity)
	if err != nil { fmt.Printf("CREATE_ACTIVITY: Failed to return the new activity: %s", err); return nil, errors.New("Failed to return the new activity") }

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


	return jsonAsBytes, nil
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
