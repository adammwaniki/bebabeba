# Vehicle Service

## How To Test API Using Postman Collection

Copy and import the following json script into your Postman collections and then run.

The variables you'll need to set in your postman environment are at the end of the json script.

**NB:** A user needs to be logged in for a session access_token to be present in order to access these endpoints.
Therefore, ensure you run the register and login scripts from the user service postman collection prior to running this one.

```json
{
  "info": {
    "name": "Vehicle Service API Testing",
    "_postman_id": "c8b27f9d-1234-4f5a-a5a1-abcdef123456",
    "description": "Full collection for Vehicle API including Vehicles, Vehicle Types management",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Create Vehicle",
      "request": {
        "method": "POST",
        "header": [
          { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
          { "key": "Content-Type", "value": "application/json", "type": "text" }
        ],
        "url": {
          "raw": "{{base_url}}/transport/vehicles",
          "host": ["{{base_url}}"],
          "path": ["transport","vehicles"]
        },
        "body": {
          "mode": "raw",
          "raw": "{\n  \"make\": \"Mazda\",\n  \"model\": \"Atenza\",\n  \"color\": \"Blue\",\n  \"year\": 2018,\n  \"seating_capacity\": 14,\n  \"vehicle_type_id\": \"2\",\n  \"license_plate\": \"KDB 123X\",\n  \"fuel_type\": 1,\n  \"engine_number\": \"ENG14523\",\n  \"chassis_number\": \"CHS14523\",\n  \"registration_date\": {\"seconds\": 1641772800, \"nanos\": 0},\n  \"insurance_expiry\": {\"seconds\": 1736073600, \"nanos\": 0}\n}"
        }
      },
      "response": []
    },
    {
      "name": "Get Vehicle by ID",
      "event": [
        {
          "listen": "prerequest",
          "script": {
            "exec": [
              "// Set vehicle_id dynamically from license plate",
              "pm.sendRequest({",
              "    url: pm.environment.get('base_url') + '/transport/vehicles?license_plate=KDB 123X',",
              "    method: 'GET',",
              "    header: { 'Authorization': 'Bearer ' + pm.environment.get('access_token') }",
              "}, function(err, res) {",
              "    if (!err && res.code === 200) {",
              "        let vehicles = res.json().vehicles;",
              "        if (vehicles.length > 0) {",
              "            pm.environment.set('vehicle_id', vehicles[0].id);",
              "        }",
              "    }",
              "});"
            ]
          }
        }
      ],
      "request": {
        "method": "GET",
        "header": [
          { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
        ],
        "url": {
          "raw": "{{base_url}}/transport/vehicles/{{vehicle_id}}",
          "host": ["{{base_url}}"],
          "path": ["transport","vehicles","{{vehicle_id}}"]
        }
      },
      "response": []
    },
    {
      "name": "Update Vehicle",
      "request": {
        "method": "PUT",
        "header": [
          { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
          { "key": "Content-Type", "value": "application/json", "type": "text" }
        ],
        "url": {
          "raw": "{{base_url}}/transport/vehicles/{{vehicle_id}}",
          "host": ["{{base_url}}"],
          "path": ["transport","vehicles","{{vehicle_id}}"]
        },
        "body": {
          "mode": "raw",
          "raw": "{\n  \"vehicle\": {\n    \"make\": \"Toyota\",\n    \"model\": \"Hiace Super\",\n    \"color\": \"White\",\n    \"year\": 2019,\n    \"seating_capacity\": 16,\n    \"fuel_type\": 1,\n    \"engine_number\": \"ENG12345\",\n    \"chassis_number\": \"CHS12345\"\n  },\n  \"update_mask\": [\"make\",\"model\",\"color\",\"year\",\"seating_capacity\",\"fuel_type\",\"engine_number\",\"chassis_number\"]\n}"
        }
      },
      "response": []
    },
    {
      "name": "Patch Vehicle Status",
      "request": {
        "method": "PATCH",
        "header": [
          { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
          { "key": "Content-Type", "value": "application/json", "type": "text" }
        ],
        "url": {
          "raw": "{{base_url}}/transport/vehicles/{{vehicle_id}}/status",
          "host": ["{{base_url}}"],
          "path": ["transport","vehicles","{{vehicle_id}}","status"]
        },
        "body": {
          "mode": "raw",
          "raw": "{\n  \"status\": \"MAINTENANCE\"\n}"
        }
      },
      "response": []
    },
    {
      "name": "Get Vehicle by License Plate",
      "request": {
        "method": "GET",
        "header": [
          { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
        ],
        "url": {
          "raw": "{{base_url}}/transport/vehicles?license_plate=KDB 123X",
          "host": ["{{base_url}}"],
          "path": ["transport","vehicles"],
          "query": [
            { "key": "license_plate", "value": "KDB 123X" }
          ]
        }
      },
      "response": []
    },
    {
      "name": "List Vehicles",
      "request": {
        "method": "GET",
        "header": [
          { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
        ],
        "url": {
          "raw": "{{base_url}}/transport/vehicles",
          "host": ["{{base_url}}"],
          "path": ["transport","vehicles"]
        }
      },
      "response": []
    },
    {
      "name": "Delete Vehicle",
      "request": {
        "method": "DELETE",
        "header": [
          { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
        ],
        "url": {
          "raw": "{{base_url}}/transport/vehicles/{{vehicle_id}}",
          "host": ["{{base_url}}"],
          "path": ["transport","vehicles","{{vehicle_id}}"]
        }
      },
      "response": []
    }
  ],
  "variable": [
    { "key": "base_url", "value": "http://localhost:8080/api/v1" },
    { "key": "access_token", "value": "" },
    { "key": "vehicle_id", "value": "" }
  ]
}
```
