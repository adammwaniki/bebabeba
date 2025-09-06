# Staff Service

## How To Test API Using Postman Collection

Copy and import the following json script into your Postman collections and then run.

The variables you'll need to set in your postman environment are at the end of the json script.

**NB:** A user needs to be logged in for a session access_token to be present in order to access these endpoints.
In this one you're lucky I finally figured out how to add the login and logout scripts.

```json
{
  "info": {
    "name": "Staff Service API Testing",
    "_postman_id": "d4b27f9e-5678-4f5a-b6b2-cdefab789012",
    "description": "Comprehensive testing collection for staff management including drivers, certifications, and license verification",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Driver Management",
      "item": [
        {
          "name": "Login",
          "request": {
            "method": "POST",
            "header": [
              { "key": "Content-Type", "value": "application/json" }
            ],
            "body": {
              "mode": "raw",
              "raw": "{\n  \"email\": \"{{user_email}}\",\n  \"password\": \"{{user_password}}\"\n}"
            },
            "url": {
              "raw": "{{base_url}}/auth/login",
              "host": ["{{base_url}}"],
              "path": ["auth", "login"]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Login successful', function() {",
                  "    pm.response.to.have.status(200);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('user');",
                  "    pm.expect(response).to.have.property('token_data');",
                  "    pm.expect(response).to.have.property('session_id');",
                  "    ",
                  "    pm.environment.set('access_token', response.token_data.access_token);",
                  "    pm.environment.set('refresh_token', response.token_data.refresh_token);",
                  "    pm.environment.set('session_id', response.session_id);",
                  "    pm.environment.set('user_id', response.user.id);",
                  "    pm.environment.set('user_email', response.user.email);",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "Create Driver",
          "request": {
            "method": "POST",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
              { "key": "Content-Type", "value": "application/json", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers"]
            },
            "body": {
              "mode": "raw",
              "raw": "{\n  \"user_id\": \"{{user_id}}\",\n  \"license_number\": \"DL1234589\",\n  \"license_class\": 2,\n  \"license_expiry\":{ \"seconds\":1767225600, \"nanos\":0 },\n  \"experience_years\": 5,\n  \"phone_number\": \"+254701234567\",\n  \"emergency_contact_name\": \"Jane Doe\",\n  \"emergency_contact_phone\": \"+254701234568\",\n  \"hire_date\":{ \"seconds\":1705238400, \"nanos\":0 }\n}"
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Driver created successfully', function() {",
                  "    pm.response.to.have.status(201);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('driver');",
                  "    pm.expect(response.driver).to.have.property('id');",
                  "    pm.expect(response.driver.license_number).to.equal('DL12345ABC');",
                  "    ",
                  "    pm.environment.set('driver_id', response.driver.id);",
                  "    pm.environment.set('driver_license_number', response.driver.license_number);",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "Get Driver by ID",
          "request": {
            "method": "GET",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers/{{driver_id}}",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers", "{{driver_id}}"]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Driver retrieved successfully', function() {",
                  "    pm.response.to.have.status(200);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('driver');",
                  "    pm.expect(response.driver.id).to.equal(pm.environment.get('driver_id'));",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "Get Driver by User ID",
          "request": {
            "method": "GET",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/users/{{user_id}}/driver",
              "host": ["{{base_url}}"],
              "path": ["users", "{{user_id}}", "driver"]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Driver retrieved by user ID successfully', function() {",
                  "    pm.response.to.have.status(200);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('driver');",
                  "    pm.expect(response.driver.user_id).to.equal(pm.environment.get('user_id'));",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "List Drivers",
          "request": {
            "method": "GET",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers?page_size=10",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers"],
              "query": [
                { "key": "page_size", "value": "10" },
                { "key": "status", "value": "ACTIVE", "disabled": true },
                { "key": "license_class", "value": "CLASS_B", "disabled": true },
                { "key": "license_expiring_soon", "value": "true", "disabled": true }
              ]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Drivers listed successfully', function() {",
                  "    pm.response.to.have.status(200);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('drivers');",
                  "    pm.expect(response.drivers).to.be.an('array');",
                  "    pm.expect(response).to.have.property('count');",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "Update Driver Status",
          "request": {
            "method": "PATCH",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
              { "key": "Content-Type", "value": "application/json", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers/{{driver_id}}/status",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers", "{{driver_id}}", "status"]
            },
            "body": {
              "mode": "raw",
              "raw": "{\n  \"status\": \"ACTIVE\",\n  \"reason\": \"License was valid blablabla.\"\n}"
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Driver status updated successfully', function() {",
                  "    pm.response.to.have.status(200);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('driver');",
                  "    pm.expect(response.driver.status).to.equal('ACTIVE');",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        }
      ]
    },
    {
      "name": "Driver Queries",
      "item": [
        {
          "name": "Get Active Drivers",
          "request": {
            "method": "GET",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers/active?page_size=10",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers", "active"],
              "query": [
                { "key": "page_size", "value": "10" },
                { "key": "license_class", "value": "CLASS_B", "disabled": true }
              ]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Active drivers retrieved successfully', function() {",
                  "    pm.response.to.have.status(200);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('drivers');",
                  "    pm.expect(response.drivers).to.be.an('array');",
                  "    // All drivers should have ACTIVE status",
                  "    response.drivers.forEach(driver => {",
                  "        pm.expect(driver.status).to.equal('ACTIVE');",
                  "    });",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "Get Expiring Licenses",
          "request": {
            "method": "GET",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers/expiring-licenses?days_ahead=30&page_size=10",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers", "expiring-licenses"],
              "query": [
                { "key": "days_ahead", "value": "30" },
                { "key": "page_size", "value": "10" }
              ]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Expiring licenses retrieved successfully', function() {",
                  "    pm.response.to.have.status(200);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('drivers');",
                  "    pm.expect(response.drivers).to.be.an('array');",
                  "    pm.expect(response).to.have.property('count');",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        }
      ]
    },
    {
      "name": "Driver Verification",
      "item": [
        {
          "name": "Verify Driver License",
          "request": {
            "method": "POST",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
              { "key": "Content-Type", "value": "application/json", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers/{{driver_id}}/verify-license",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers", "{{driver_id}}", "verify-license"]
            },
            "body": {
              "mode": "raw",
              "raw": "{\n  \"license_number\": \"{{driver_license_number}}\"\n}"
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('License verification completed', function() {",
                  "    pm.response.to.have.status(200);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('verification_result');",
                  "    pm.expect(response).to.have.property('verified_at');",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        }
      ]
    },
    {
      "name": "Driver Certifications",
      "item": [
        {
          "name": "Add Driver Certification",
          "request": {
            "method": "POST",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
              { "key": "Content-Type", "value": "application/json", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers/{{driver_id}}/certifications",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers", "{{driver_id}}", "certifications"]
            },
            "body": {
              "mode": "raw",
              "raw": "{\n  \"certification_name\": \"Defensive Driving Certificate\",\n  \"issued_by\": \"Kenya National Transport Safety Authority\",\n  \"issue_date\": { \"seconds\":1705276800, \"nanos\":0 },\n  \"expiry_date\": { \"seconds\":1736899200, \"nanos\":0 }\n}"
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Certification added successfully', function() {",
                  "    pm.response.to.have.status(201);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('certification');",
                  "    pm.expect(response.certification).to.have.property('id');",
                  "    pm.expect(response.certification.certification_name).to.equal('Defensive Driving Certificate');",
                  "    ",
                  "    pm.environment.set('certification_id', response.certification.id);",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "List Driver Certifications",
          "request": {
            "method": "GET",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers/{{driver_id}}/certifications?page_size=10",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers", "{{driver_id}}", "certifications"],
              "query": [
                { "key": "page_size", "value": "10" },
                { "key": "status", "value": "CERT_ACTIVE", "disabled": true },
                { "key": "expiring_soon", "value": "true", "disabled": true }
              ]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Driver certifications retrieved successfully', function() {",
                  "    pm.response.to.have.status(200);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('certifications');",
                  "    pm.expect(response.certifications).to.be.an('array');",
                  "    pm.expect(response).to.have.property('count');",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        }
      ]
    },
    {
      "name": "Test Scenarios",
      "item": [
        {
          "name": "Create Driver with Invalid License Class",
          "request": {
            "method": "POST",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
              { "key": "Content-Type", "value": "application/json", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers"]
            },
            "body": {
              "mode": "raw",
              "raw": "{\n  \"user_id\": \"{{user_id}}\",\n  \"license_number\": \"DL67890XYZ\",\n  \"license_class\": 0,\n  \"license_expiry\": { \"seconds\":1767139200, \"nanos\":0 },\n  \"experience_years\": 3,\n  \"phone_number\": \"+254701234569\",\n  \"emergency_contact_name\": \"John Smith\",\n  \"emergency_contact_phone\": \"+254701234570\"\n}"
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Invalid license class rejected', function() {",
                  "    pm.response.to.have.status(400);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('error');",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "Get Non-existent Driver",
          "request": {
            "method": "GET",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers/00000000-0000-0000-0000-000000000000",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers", "00000000-0000-0000-0000-000000000000"]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Non-existent driver returns 404', function() {",
                  "    pm.response.to.have.status(404);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('error');",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "Update Status with Invalid Status",
          "request": {
            "method": "PATCH",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
              { "key": "Content-Type", "value": "application/json", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers/{{driver_id}}/status",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers", "{{driver_id}}", "status"]
            },
            "body": {
              "mode": "raw",
              "raw": "{\n  \"status\": \"INVALID_STATUS\",\n  \"reason\": \"Testing invalid status\"\n}"
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Invalid status rejected', function() {",
                  "    pm.response.to.have.status(400);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('error');",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        },
        {
          "name": "Unauthorized Access Test",
          "request": {
            "method": "GET",
            "header": [],
            "url": {
              "raw": "{{base_url}}/transport/drivers",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers"]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Unauthorized access denied', function() {",
                  "    pm.response.to.have.status(401);",
                  "});"
                ],
                "type": "text/javascript"
              }
            }
          ]
        }
      ]
    },
    {
      "name": "Integration Flow",
      "item": [
        {
          "name": "Complete Driver Onboarding Flow",
          "event": [
            {
              "listen": "prerequest",
              "script": {
                "exec": [
                  "// This test demonstrates a complete driver onboarding workflow",
                  "console.log('Starting complete driver onboarding flow...');"
                ]
              }
            }
          ],
          "request": {
            "method": "POST",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
              { "key": "Content-Type", "value": "application/json", "type": "text" }
            ],
            "url": {
              "raw": "{{base_url}}/transport/drivers",
              "host": ["{{base_url}}"],
              "path": ["transport", "drivers"]
            },
            "body": {
              "mode": "raw",
              "raw": "{\n  \"user_id\": \"{{user_id}}\",\n  \"license_number\": \"DL2932{{$randomInt}}\",\n  \"license_class\": 2,\n  \"license_expiry\":{ \"seconds\":1782777600, \"nanos\":0 },\n  \"experience_years\": 8,\n  \"phone_number\": \"0701{{$randomInt}}\",\n  \"emergency_contact_name\": \"Emergency Contact\",\n  \"emergency_contact_phone\": \"+254702{{$randomInt}}\",\n  \"hire_date\":{ \"seconds\":1725148800, \"nanos\":0 }\n}"
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Driver onboarding flow - Driver created', function() {",
                  "    pm.response.to.have.status(201);",
                  "    const response = pm.response.json();",
                  "    pm.expect(response).to.have.property('driver');",
                  "    pm.environment.set('onboarding_driver_id', response.driver.id);",
                  "    pm.environment.set('onboarding_license_number', response.driver.license_number);",
                  "});",
                  "",
                  "// Step 2: Verify the license",
                  "if (pm.response.code === 201) {",
                  "    const driverId = pm.environment.get('onboarding_driver_id');",
                  "    const licenseNumber = pm.environment.get('onboarding_license_number');",
                  "    ",
                  "    pm.sendRequest({",
                  "        url: pm.environment.get('base_url') + '/transport/drivers/' + driverId + '/verify-license',",
                  "        method: 'POST',",
                  "        header: {",
                  "            'Authorization': 'Bearer ' + pm.environment.get('access_token'),",
                  "            'Content-Type': 'application/json'",
                  "        },",
                  "        body: {",
                  "            mode: 'raw',",
                  "            raw: JSON.stringify({ license_number: licenseNumber })",
                  "        }",
                  "    }, function(err, res) {",
                  "        pm.test('Driver onboarding flow - License verified', function() {",
                  "            pm.expect(res.code).to.equal(200);",
                  "        });",
                  "        ",
                  "        // Step 3: Activate the driver",
                  "        pm.sendRequest({",
                  "            url: pm.environment.get('base_url') + '/transport/drivers/' + driverId + '/status',",
                  "            method: 'PATCH',",
                  "            header: {",
                  "                'Authorization': 'Bearer ' + pm.environment.get('access_token'),",
                  "                'Content-Type': 'application/json'",
                  "            },",
                  "            body: {",
                  "                mode: 'raw',",
                  "                raw: JSON.stringify({",
                  "                    status: 'ACTIVE',",
                  "                    reason: 'Onboarding completed successfully'",
                  "                })",
                  "            }",
                  "        }, function(err, res) {",
                  "            pm.test('Driver onboarding flow - Driver activated', function() {",
                  "                pm.expect(res.code).to.equal(200);",
                  "                const response = res.json();",
                  "                pm.expect(response.driver.status).to.equal('ACTIVE');",
                  "            });",
                  "        });",
                  "    });",
                  "}"
                ],
                "type": "text/javascript"
              }
            }
          ]
        }
      ]
    },
    {
      "name": "Session Management",
      "item": [
        {
          "name": "Logout",
          "request": {
            "method": "POST",
            "header": [
              { "key": "Authorization", "value": "Bearer {{access_token}}", "type": "text" },
              { "key": "Content-Type", "value": "application/json", "type": "text" }
            ],
            "body": {
              "mode": "raw",
              "raw": "{\n  \"refresh_token\": \"{{refresh_token}}\"\n}"
            },
            "url": {
              "raw": "{{base_url}}/auth/logout",
              "host": ["{{base_url}}"],
              "path": ["auth", "logout"]
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Logout successful', function() {",
                  "    pm.response.to.have.status(200);",
                  "});",
                  "",
                  "// Clear tokens after successful logout",
                  "if (pm.response.code === 200) {",
                  "    pm.environment.unset('access_token');",
                  "    pm.environment.unset('refresh_token');",
                  "    pm.environment.unset('session_id');",
                  "    console.log('Authentication tokens cleared');",
                  "}"
                ],
                "type": "text/javascript"
              }
            }
          ]
        }
      ]
    }
  ],
  "variable": [
    { "key": "base_url", "value": "http://localhost:8080/api/v1" },
    { "key": "access_token", "value": "" },
    { "key": "refresh_token", "value": "" },
    { "key": "session_id", "value": "" },
    { "key": "user_id", "value": "" },
    { "key": "user_email", "value": "" },
    { "key": "user_password", "value": "securepassword123" },
    { "key": "driver_id", "value": "" },
    { "key": "driver_license_number", "value": "" },
    { "key": "certification_id", "value": "" },
    { "key": "onboarding_driver_id", "value": "" },
    { "key": "onboarding_license_number", "value": "" }
  ]
}
```
