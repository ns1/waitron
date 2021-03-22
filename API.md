


# Waitron
Endpoints for server provisioning
  

## Informations

### Version

2

### Contact

  

## Content negotiation

### URI Schemes
  * http

### Consumes
  * application/json

### Produces
  * application/json

## All endpoints

###  operations

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| GET | /cancel/{hostname}/{token} | [get cancel hostname token](#get-cancel-hostname-token) |  |
| GET | /cleanhistory | [get cleanhistory](#get-cleanhistory) |  |
| GET | /definition/{hostname}/{type} | [get definition hostname type](#get-definition-hostname-type) |  |
| GET | /done/{hostname}/{token} | [get done hostname token](#get-done-hostname-token) |  |
| GET | /health | [get health](#get-health) |  |
| GET | /job/{token} | [get job token](#get-job-token) |  |
| GET | /status | [get status](#get-status) |  |
| GET | /status/{hostname} | [get status hostname](#get-status-hostname) |  |
| GET | /template/{template}/{hostname}/{token} | [get template template hostname token](#get-template-template-hostname-token) |  |
| GET | /v1/boot/{macaddr} | [get v1 boot macaddr](#get-v1-boot-macaddr) |  |
| PUT | /build/{hostname}/{type} | [put build hostname type](#put-build-hostname-type) |  |
  


## Paths

### <span id="get-cancel-hostname-token"></span> get cancel hostname token (*GetCancelHostnameToken*)

```
GET /cancel/{hostname}/{token}
```

Remove the server from build mode

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| hostname | `path` | string | `string` |  | ✓ |  | Hostname |
| token | `path` | string | `string` |  | ✓ |  | Token |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-cancel-hostname-token-200) | OK | {"State": "OK"} |  | [schema](#get-cancel-hostname-token-200-schema) |
| [500](#get-cancel-hostname-token-500) | Internal Server Error | Failed to cancel build mode |  | [schema](#get-cancel-hostname-token-500-schema) |

#### Responses


##### <span id="get-cancel-hostname-token-200"></span> 200 - {"State": "OK"}
Status: OK

###### <span id="get-cancel-hostname-token-200-schema"></span> Schema
   
  



##### <span id="get-cancel-hostname-token-500"></span> 500 - Failed to cancel build mode
Status: Internal Server Error

###### <span id="get-cancel-hostname-token-500-schema"></span> Schema
   
  



### <span id="get-cleanhistory"></span> get cleanhistory (*GetCleanhistory*)

```
GET /cleanhistory
```

Clear all completed jobs from the in-memory history of Waitron

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-cleanhistory-200) | OK | {"State": "OK"} |  | [schema](#get-cleanhistory-200-schema) |
| [500](#get-cleanhistory-500) | Internal Server Error | Failed to clean history |  | [schema](#get-cleanhistory-500-schema) |

#### Responses


##### <span id="get-cleanhistory-200"></span> 200 - {"State": "OK"}
Status: OK

###### <span id="get-cleanhistory-200-schema"></span> Schema
   
  



##### <span id="get-cleanhistory-500"></span> 500 - Failed to clean history
Status: Internal Server Error

###### <span id="get-cleanhistory-500-schema"></span> Schema
   
  



### <span id="get-definition-hostname-type"></span> get definition hostname type (*GetDefinitionHostnameType*)

```
GET /definition/{hostname}/{type}
```

Return the waitron configuration details for a machine

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| hostname | `path` | string | `string` |  | ✓ |  | Hostname |
| type | `path` | string | `string` |  | ✓ |  | Build Type |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-definition-hostname-type-200) | OK | Machine config in JSON format. |  | [schema](#get-definition-hostname-type-200-schema) |
| [404](#get-definition-hostname-type-404) | Not Found | Machine not found |  | [schema](#get-definition-hostname-type-404-schema) |

#### Responses


##### <span id="get-definition-hostname-type-200"></span> 200 - Machine config in JSON format.
Status: OK

###### <span id="get-definition-hostname-type-200-schema"></span> Schema
   
  



##### <span id="get-definition-hostname-type-404"></span> 404 - Machine not found
Status: Not Found

###### <span id="get-definition-hostname-type-404-schema"></span> Schema
   
  



### <span id="get-done-hostname-token"></span> get done hostname token (*GetDoneHostnameToken*)

```
GET /done/{hostname}/{token}
```

Remove the server from build mode

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| hostname | `path` | string | `string` |  | ✓ |  | Hostname |
| token | `path` | string | `string` |  | ✓ |  | Token |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-done-hostname-token-200) | OK | {"State": "OK"} |  | [schema](#get-done-hostname-token-200-schema) |
| [500](#get-done-hostname-token-500) | Internal Server Error | Failed to finish build mode |  | [schema](#get-done-hostname-token-500-schema) |

#### Responses


##### <span id="get-done-hostname-token-200"></span> 200 - {"State": "OK"}
Status: OK

###### <span id="get-done-hostname-token-200-schema"></span> Schema
   
  



##### <span id="get-done-hostname-token-500"></span> 500 - Failed to finish build mode
Status: Internal Server Error

###### <span id="get-done-hostname-token-500-schema"></span> Schema
   
  



### <span id="get-health"></span> get health (*GetHealth*)

```
GET /health
```

Check that Waitron is running

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-health-200) | OK | {"State": "OK"} |  | [schema](#get-health-200-schema) |

#### Responses


##### <span id="get-health-200"></span> 200 - {"State": "OK"}
Status: OK

###### <span id="get-health-200-schema"></span> Schema
   
  



### <span id="get-job-token"></span> get job token (*GetJobToken*)

```
GET /job/{token}
```

Return details for the specified job token

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| token | `path` | string | `string` |  | ✓ |  | Token |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-job-token-200) | OK | Job details in JSON format. |  | [schema](#get-job-token-200-schema) |
| [404](#get-job-token-404) | Not Found | Job not found |  | [schema](#get-job-token-404-schema) |

#### Responses


##### <span id="get-job-token-200"></span> 200 - Job details in JSON format.
Status: OK

###### <span id="get-job-token-200-schema"></span> Schema
   
  



##### <span id="get-job-token-404"></span> 404 - Job not found
Status: Not Found

###### <span id="get-job-token-404-schema"></span> Schema
   
  



### <span id="get-status"></span> get status (*GetStatus*)

```
GET /status
```

Dictionary with jobs and status

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-status-200) | OK | Dictionary with jobs and status |  | [schema](#get-status-200-schema) |
| [500](#get-status-500) | Internal Server Error | The error encountered |  | [schema](#get-status-500-schema) |

#### Responses


##### <span id="get-status-200"></span> 200 - Dictionary with jobs and status
Status: OK

###### <span id="get-status-200-schema"></span> Schema
   
  



##### <span id="get-status-500"></span> 500 - The error encountered
Status: Internal Server Error

###### <span id="get-status-500-schema"></span> Schema
   
  



### <span id="get-status-hostname"></span> get status hostname (*GetStatusHostname*)

```
GET /status/{hostname}
```

Build status of the server

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| hostname | `path` | string | `string` |  | ✓ |  | Hostname |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-status-hostname-200) | OK | The status: (installing or installed) |  | [schema](#get-status-hostname-200-schema) |
| [404](#get-status-hostname-404) | Not Found | Failed to find active job for host |  | [schema](#get-status-hostname-404-schema) |

#### Responses


##### <span id="get-status-hostname-200"></span> 200 - The status: (installing or installed)
Status: OK

###### <span id="get-status-hostname-200-schema"></span> Schema
   
  



##### <span id="get-status-hostname-404"></span> 404 - Failed to find active job for host
Status: Not Found

###### <span id="get-status-hostname-404-schema"></span> Schema
   
  



### <span id="get-template-template-hostname-token"></span> get template template hostname token (*GetTemplateTemplateHostnameToken*)

```
GET /template/{template}/{hostname}/{token}
```

Render either the finish or the preseed template

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| hostname | `path` | string | `string` |  | ✓ |  | Hostname |
| template | `path` | string | `string` |  | ✓ |  | The template to be rendered |
| token | `path` | string | `string` |  | ✓ |  | Token |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-template-template-hostname-token-200) | OK | Rendered template |  | [schema](#get-template-template-hostname-token-200-schema) |
| [400](#get-template-template-hostname-token-400) | Bad Request | Unable to render template |  | [schema](#get-template-template-hostname-token-400-schema) |

#### Responses


##### <span id="get-template-template-hostname-token-200"></span> 200 - Rendered template
Status: OK

###### <span id="get-template-template-hostname-token-200-schema"></span> Schema
   
  



##### <span id="get-template-template-hostname-token-400"></span> 400 - Unable to render template
Status: Bad Request

###### <span id="get-template-template-hostname-token-400-schema"></span> Schema
   
  



### <span id="get-v1-boot-macaddr"></span> get v1 boot macaddr (*GetV1BootMacaddr*)

```
GET /v1/boot/{macaddr}
```

Dictionary with kernel, intrd(s) and commandline for pixiecore

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| macaddr | `path` | string | `string` |  | ✓ |  | MacAddress |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-v1-boot-macaddr-200) | OK | Dictionary with kernel, intrd(s) and commandline for pixiecore |  | [schema](#get-v1-boot-macaddr-200-schema) |
| [500](#get-v1-boot-macaddr-500) | Internal Server Error | failed to get pxe config |  | [schema](#get-v1-boot-macaddr-500-schema) |

#### Responses


##### <span id="get-v1-boot-macaddr-200"></span> 200 - Dictionary with kernel, intrd(s) and commandline for pixiecore
Status: OK

###### <span id="get-v1-boot-macaddr-200-schema"></span> Schema
   
  



##### <span id="get-v1-boot-macaddr-500"></span> 500 - failed to get pxe config
Status: Internal Server Error

###### <span id="get-v1-boot-macaddr-500-schema"></span> Schema
   
  



### <span id="put-build-hostname-type"></span> put build hostname type (*PutBuildHostnameType*)

```
PUT /build/{hostname}/{type}
```

Put the server in build mode

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| hostname | `path` | string | `string` |  | ✓ |  | Hostname |
| type | `path` | string | `string` |  | ✓ |  | Build Type |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#put-build-hostname-type-200) | OK | {"State": "OK", "Token": <UUID of the build>} |  | [schema](#put-build-hostname-type-200-schema) |
| [500](#put-build-hostname-type-500) | Internal Server Error | Failed to set build mode on hostname |  | [schema](#put-build-hostname-type-500-schema) |

#### Responses


##### <span id="put-build-hostname-type-200"></span> 200 - {"State": "OK", "Token": <UUID of the build>}
Status: OK

###### <span id="put-build-hostname-type-200-schema"></span> Schema
   
  



##### <span id="put-build-hostname-type-500"></span> 500 - Failed to set build mode on hostname
Status: Internal Server Error

###### <span id="put-build-hostname-type-500-schema"></span> Schema
   
  



## Models
