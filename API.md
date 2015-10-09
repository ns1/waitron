
# Waitron
Templates for server provisioning

Table of Contents

1. [List machines handled by waitron](#list)
1. [Dictionary with machines and its status](#status)
1. [Dictionary with kernel, intrd(s) and commandline for pixiecore](#v1)
1. [Renders either the finish or the preseed template](#{hostname})

<a name="list"></a>

## list

| Specification | Value |
|-----|-----|
| Resource Path | /list |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /list | [GET](#listMachinesHandler) | List machines handled by waitron |



<a name="listMachinesHandler"></a>

#### API: /list (GET)


List machines handled by waitron



| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | string | List of machines |
| 500 | object | string | Unable to list machines |




### Models


<a name="status"></a>

## status

| Specification | Value |
|-----|-----|
| Resource Path | /status |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /status | [GET](#status) | Dictionary with machines and its status |



<a name="status"></a>

#### API: /status (GET)


Dictionary with machines and its status



| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | Dictionary with machines and its status |




### Models


<a name="v1"></a>

## v1

| Specification | Value |
|-----|-----|
| Resource Path | /v1 |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /v1/boot/\{macaddr\} | [GET](#pixieHandler) | Dictionary with kernel, intrd(s) and commandline for pixiecore |



<a name="pixieHandler"></a>

#### API: /v1/boot/\{macaddr\} (GET)


Dictionary with kernel, intrd(s) and commandline for pixiecore



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| macaddr | path | string | MacAddress | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | Dictionary with kernel, intrd(s) and commandline for pixiecore |
| 404 | object | string | Not in build mode |
| 500 | object | string | Unable to find host definition for hostname |




### Models


<a name="{hostname}"></a>

## {hostname}

| Specification | Value |
|-----|-----|
| Resource Path | /{hostname} |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| \{hostname\}/\{template\}/\{token\} | [GET](#templateHandler) | Renders either the finish or the preseed template |
| /\{hostname\}/build | [GET](#buildHandler) | Put the server in build mode |
| /\{hostname\}/done/\{token\} | [GET](#doneHandler) | Removes the server from build mode |
| /\{hostname\}/status | [GET](#hostStatus) | Build status of the server |



<a name="templateHandler"></a>

#### API: \{hostname\}/\{template\}/\{token\} (GET)


Renders either the finish or the preseed template



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| hostname | path | string | Hostname | Yes |
| template | path | string | The template to be rendered | Yes |
| token | path | string | Token | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | Rendered template |
| 400 | object | string | Unable to find host definition for hostname |
| 400 | object | string | Unable to render template |
| 401 | object | string | Invalid token |


<a name="buildHandler"></a>

#### API: /\{hostname\}/build (GET)


Put the server in build mode



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| hostname | path | string | Hostname | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | OK |
| 500 | object | string | Unable to find host definition for hostname |
| 500 | object | string | Failed to set build mode on hostname |


<a name="doneHandler"></a>

#### API: /\{hostname\}/done/\{token\} (GET)


Removes the server from build mode



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| hostname | path | string | Hostname | Yes |
| token | path | string | Token | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | OK |
| 500 | object | string | Unable to find host definition for hostname |
| 500 | object | string | Failed to cancel build mode |
| 401 | object | string | Invalid token |


<a name="hostStatus"></a>

#### API: /\{hostname\}/status (GET)


Build status of the server



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| hostname | path | string | Hostname | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | The status: (installing or installed) |
| 500 | object | string | Unknown state |




### Models


