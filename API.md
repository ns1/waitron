
# Waitron
Templates for server provisioning

Table of Contents

1. [Put the server in build mode](#build)
1. [Removes the server from build mode and runs post-build commands related to requested build terminations.](#cancel)
1. [Removes the server from build mode and runs post-build comands related to normal install completion.](#done)
1. [health](#health)
1. [List machines handled by waitron](#list)
1. [Build status of the server](#status)
1. [Renders either the finish or the preseed template](#template)
1. [Dictionary with kernel, intrd(s) and commandline for pixiecore](#v1)

<a name="build"></a>

## build

| Specification | Value |
|-----|-----|
| Resource Path | /build |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| build/\{hostname\} | [PUT](#buildHandler) | Put the server in build mode |



<a name="buildHandler"></a>

#### API: build/\{hostname\} (PUT)


Put the server in build mode



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| hostname | path | string | Hostname | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | {"State": "OK", "Token": <UUID of the build>} |
| 500 | object | string | Unable to find host definition for hostname |
| 500 | object | string | Failed to set build mode on hostname |


<a name="cancel"></a>

## cancel

| Specification | Value |
|-----|-----|
| Resource Path | /cancel |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /cancel/\{hostname\}/\{token\} | [GET](#cancelHandler) | Removes the server from build mode and runs post-build commands related to requested build terminations. |



<a name="cancelHandler"></a>

#### API: /cancel/\{hostname\}/\{token\} (GET)


Removes the server from build mode and runs post-build commands related to requested build terminations.



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| hostname | path | string | Hostname | Yes |
| token | path | string | Token | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | {"State": "OK"} |
| 500 | object | string | Failed to cancel build mode |
| 400 | object | string | Not in build mode or definition does not exist |
| 401 | object | string | Invalid token |


<a name="done"></a>

## done

| Specification | Value |
|-----|-----|
| Resource Path | /done |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /done/\{hostname\}/\{token\} | [GET](#doneHandler) | Removes the server from build mode and runs post-build comands related to normal install completion. |



<a name="doneHandler"></a>

#### API: /done/\{hostname\}/\{token\} (GET)


Removes the server from build mode and runs post-build comands related to normal install completion.



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| hostname | path | string | Hostname | Yes |
| token | path | string | Token | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | {"State": "OK"} |
| 500 | object | string | Failed to finish build mode |
| 400 | object | string | Not in build mode or definition does not exist |
| 401 | object | string | Invalid token |


<a name="health"></a>

## health

| Specification | Value |
|-----|-----|
| Resource Path | /health |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /health | [GET](#healthHandler) |  |



<a name="healthHandler"></a>

#### API: /health (GET)






| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | {"State": "OK"} |


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
| /status/\{hostname\} | [GET](#hostStatus) | Build status of the server |
| /status | [GET](#status) | Dictionary with machines and its status |



<a name="hostStatus"></a>

#### API: /status/\{hostname\} (GET)


Build status of the server



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| hostname | path | string | Hostname | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | The status: (installing or installed) |
| 500 | object | string | Unknown state |


<a name="status"></a>

#### API: /status (GET)


Dictionary with machines and its status



| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | Dictionary with machines and its status |


<a name="template"></a>

## template

| Specification | Value |
|-----|-----|
| Resource Path | /template |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /template/\{template\}/\{hostname\}/\{token\} | [GET](#templateHandler) | Renders either the finish or the preseed template |



<a name="templateHandler"></a>

#### API: /template/\{template\}/\{hostname\}/\{token\} (GET)


Renders either the finish or the preseed template



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| hostname | path | string | Hostname | Yes |
| template | path | string | The template to be rendered | Yes |
| token | path | string | Token | Yes |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | string | Rendered template |
| 400 | object | string | Not in build mode or definition does not exist |
| 400 | object | string | Unable to render template |
| 401 | object | string | Invalid token |


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


