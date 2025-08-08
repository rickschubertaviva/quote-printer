package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"slices"

	"github.com/TylerBrock/colorjson"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"
	"github.com/manifoldco/promptui"
	"github.com/niemeyer/pretty"
	"k8s.io/utils/ptr"
)

func main() {
	cliArgs := cliArguments()

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}

	dynamoDBClient := dynamodb.NewFromConfig(cfg)
	accountNumber := getAccountNumber(cfg)
	tableName := getTableName(cliArgs.scanForPolicy, accountNumber)

	if cliArgs.scanForPolicy {
		getPolicy(dynamoDBClient, tableName, cliArgs)
	} else {
		getQuote(dynamoDBClient, tableName, cliArgs)
	}
}

func getTableName(scanForPolicy bool, accountNumber string) string {
	switch accountNumber {
	// Testing
	case "596956765480":
		switch scanForPolicy {
		case true:
			return "policy-api-policyTable777C1DD9-V7ZQ8ZD0HHTM"
		default:
			return "quote-api-quoteTableC29293A1-5EAYAUNFL0XD"
		}
		// Staging
	case "743702672182":
		switch scanForPolicy {
		case true:
			return "policy-api-policyTable777C1DD9-1V1XGR44T8OX0"
		default:
			return "quote-api-quoteTableC29293A1-NW27YUGPQXFP"
		}
	}

	log.Fatalf("Unknown account number %s", accountNumber)

	return ""
}

func colorPrintJSON(unmarshalled map[string]interface{}) {
	formatter := colorjson.NewFormatter()
	formatter.Indent = 2

	formatted, err := formatter.Marshal(unmarshalled)
	if err != nil {
		log.Fatalf("Failed to color marshal JSON: %s", err)
	}

	pretty.Println(string(formatted))
}

func getItemFromDynamoDB(
	dynamoDBClient *dynamodb.Client,
	tableName string,
	pk string,
	sk string,
) map[string]interface{} {
	var unmarshalled map[string]interface{}

	params := &dynamodb.GetItemInput{
		TableName: ptr.To(tableName),
		Key: map[string]types.AttributeValue{
			"_pk": &types.AttributeValueMemberS{
				Value: pk,
			},
			"_sk": &types.AttributeValueMemberS{
				Value: sk,
			},
		},
	}

	result, err := dynamoDBClient.GetItem(context.Background(), params)
	if err != nil {
		log.Fatalf("Query API call failed: %s", err)
	}

	err = attributevalue.UnmarshalMap(result.Item, &unmarshalled)
	if err != nil {
		log.Fatalf("Failed to unmarshal DynamoDB item: %s", err)
	}

	return unmarshalled
}

func queryPolicyTableByPolicyID(
	dynamoDBClient *dynamodb.Client,
	tableName string,
	policyID string,
) *dynamodb.QueryOutput {
	params := &dynamodb.QueryInput{
		ExpressionAttributeNames: map[string]string{
			"#pk": "_pk",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pkval": &types.AttributeValueMemberS{
				Value: fmt.Sprintf("policy/%s", policyID),
			},
		},
		KeyConditionExpression: ptr.To("#pk = :pkval"),
		TableName:              ptr.To(tableName),
	}

	items, err := dynamoDBClient.Query(context.Background(), params)
	if err != nil {
		log.Fatalf("Query API call failed: %s", err)
	}

	return items
}

func getSortKeysFromAllDynamoDBItems(
	items *dynamodb.QueryOutput,
) []string {
	var sortKeys []string

	for _, item := range items.Items {
		if sk, ok := item["_sk"]; ok && sk.(*types.AttributeValueMemberS) != nil {
			sortKeys = append(sortKeys, sk.(*types.AttributeValueMemberS).Value)
		}
	}

	return orderPolicySortKeys(sortKeys)
}

func getPolicy(
	dynamoDBClient *dynamodb.Client,
	tableName string,
	cliArgs CLIArguments,
) {
	items := queryPolicyTableByPolicyID(
		dynamoDBClient,
		tableName,
		cliArgs.idToGet,
	)

	if len(items.Items) == 0 {
		log.Fatalf("No items found for ID %s in table %s", cliArgs.idToGet, tableName)
	}

	row := getItemFromDynamoDB(
		dynamoDBClient,
		tableName,
		fmt.Sprintf("policy/%s", cliArgs.idToGet),
		getSortKeyToInspect(cliArgs, items),
	)

	colorPrintJSON(row)
}

func getQuote(
	dynamoDBClient *dynamodb.Client,
	tableName string,
	cliArgs CLIArguments,
) {
	item := getItemFromDynamoDB(
		dynamoDBClient,
		tableName,
		fmt.Sprintf("quote/%s", cliArgs.idToGet),
		"quote",
	)

	colorPrintJSON(item)
}

type CLIArguments struct {
	scanForPolicy         bool
	idToGet               string
	autoSelectLatestState bool
}

func cliArguments() CLIArguments {
	var idToGet string

	if len(os.Args) < 1 {
		log.Fatalf("The final argument must be an ID (either quote or policy) which we are supposed to get")
	}

	scanForPolicy := flag.Bool("policy", false, "If active, this tool is going to query the policy table and give you some options of which item to retrieve.")
	autoSelectLatestState := flag.Bool("latest", false, "If active, this tool will automatically select the latest state of the policy instead of prompting.")
	flag.Parse()

	idToGet = os.Args[len(os.Args)-1]
	err := uuid.Validate(idToGet)
	if err != nil {
		log.Fatalf("The final argument must be a valid UUID: %s", err)
	}

	return CLIArguments{
		scanForPolicy:         *scanForPolicy,
		idToGet:               idToGet,
		autoSelectLatestState: *autoSelectLatestState,
	}
}

func getAccountNumber(cfg aws.Config) string {
	sts.NewFromConfig(cfg)

	stsClient := sts.NewFromConfig(cfg)
	output, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatalf("failed to get caller identity, %v", err)
	}

	if output.Account == nil {
		log.Fatal("No account number found in the response")
	}

	return *output.Account
}

// STATE entries on top, with STATE being the first sort key and then the latest
// one on top.
func orderPolicySortKeys(sortKeys []string) []string {
	if len(sortKeys) < 2 {
		return sortKeys
	}

	var stateKeys []string
	var nonStateKeys []string

	for _, sk := range sortKeys {
		if strings.Contains(sk, "STATE") {
			stateKeys = append(stateKeys, sk)
		} else {
			nonStateKeys = append(nonStateKeys, sk)
		}
	}

	slices.Sort(nonStateKeys)
	slices.Sort(stateKeys)

	var finalSortKeys []string
	finalSortKeys = append(finalSortKeys, stateKeys...)
	finalSortKeys = append(finalSortKeys, nonStateKeys...)

	return finalSortKeys
}

func getSortKeyToInspect(
	cliArgs CLIArguments,
	items *dynamodb.QueryOutput,
) string {
	if cliArgs.autoSelectLatestState {
		return "STATE"
	}

	sortKeyPrompt := promptui.Select{
		Label: "Select a sort key to inspect",
		Items: getSortKeysFromAllDynamoDBItems(items),
	}

	_, choice, err := sortKeyPrompt.Run()

	if err != nil || choice == "" {
		log.Fatalf("Failed to run prompt or no sort key selected: %s", err)
	}

	return choice
}
