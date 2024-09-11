# Questspace integration tests

## Requirements
 - Go >=1.22.1
 - Docker

## Running tests
To run integration tests, use this command:
```shell
go test -v -run ^TestQuestspace$ ./test/integration/...
```
*Note for testers: You can freely run this command from every subdir of test path starting from `src`, 
running it from repo root will result in failure.*

## Adding cases
All testcases should be placed in `testdata/cases` directory using pattern `{case_number}/tc.yaml`

### Case syntax
All cases are written in YAML and contain series of requests and expected responses.
For single-request cases, you can write request data on the top level:
```yaml
name: your testcase
ignore: true
method: GET
uri: /ping
expected-status: 200
```
*NOTE: setting ignore option will result in test case success without running it*
For many requests you need to state them inside `requests` list field:
```yaml
name: your testcase
requests:
  - method: GET
    uri: /ping
    expected-status: 200
  - method: GET
    uri: /ping
    expected-status: 200
```
Full request schema:
```yaml
method: string
uri: string
authorization: string
json-input: string
expected-status: int
expected-json: string
```


Also case reader supports value assignment and any(wildcard) value.

You can set unnecessary values of expected json to `"$ANY$"` if you do not want to compare it (or do not know what exactly will be in this response field).
Example:
```yaml
...
expected-json: >
  {
    "random_value": "$ANY$"
  }
```
This will match all json payloads that have single `"random_value"` field regardless of its' content.


If you need to use response data in next requests, you can use value assignment. Statement `"$SET$:VARIABLE_NAME"` will set the result value of the response to the variable `VARIABLE_NAME`.
Here the features and limitations:
 - Only uppercase ascii-letters, numbers and "_" character can be used in variable names
 - If you don't use the variable in next response, setting it is equal to stating field to `"$ANY$"`, as the field written to variable will not be compared at all
 - After response with variable set succeeds, it can be used in `uri`, `authorization`, `json-input`, and `expected-json` fields of next responses using `$VARIABLE_NAME`

Example:
```yaml
requests:
  - uri: /random
    expected-json: >
      {
        "random_value": "$SET$:RANDOM_VALUE"
      }
  - uri: /set/$RANDOM_VALUE
    authorization: $RANDOM_VALUE
    json-input: >
      {
        "password": "$RANDOM_VALUE"
      }
    expected-json: >
      {
        "changed": "$RANDOM_VALUE"
      }
```