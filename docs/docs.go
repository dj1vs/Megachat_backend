// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/coding": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "tags": [
                    "accounts"
                ],
                "summary": "Обрабатывает сообщения от сервиса кодирования",
                "parameters": [
                    {
                        "description": "Сообщение от сервиса кодирования",
                        "name": "message",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/ds.CodingResp"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    },
                    "400": {
                        "description": "Недопустимый метод"
                    },
                    "500": {
                        "description": "Ошибка при чтении JSON"
                    }
                }
            }
        },
        "/front": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "tags": [
                    "accounts"
                ],
                "summary": "Обрабатывает сообщения от фронтенда (прикладной уровень)",
                "parameters": [
                    {
                        "description": "Сообщение от фронтенда",
                        "name": "message",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/ds.FrontReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    },
                    "400": {
                        "description": "Недопустимый метод"
                    },
                    "500": {
                        "description": "Ошибка при чтении JSON"
                    }
                }
            }
        }
    },
    "definitions": {
        "ds.CodingResp": {
            "type": "object",
            "properties": {
                "payload": {
                    "$ref": "#/definitions/ds.CodingRespPayload"
                },
                "time": {
                    "type": "integer"
                },
                "username": {
                    "type": "string"
                }
            }
        },
        "ds.CodingRespPayload": {
            "type": "object",
            "properties": {
                "data": {
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "segment_cnt": {
                    "type": "integer"
                },
                "segment_num": {
                    "type": "integer"
                },
                "status": {
                    "type": "string"
                }
            }
        },
        "ds.FrontReq": {
            "type": "object",
            "properties": {
                "payload": {
                    "$ref": "#/definitions/ds.FrontReqPayload"
                },
                "time": {
                    "type": "integer"
                },
                "username": {
                    "type": "string"
                }
            }
        },
        "ds.FrontReqPayload": {
            "type": "object",
            "properties": {
                "data": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
