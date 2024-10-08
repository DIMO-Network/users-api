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
        "/": {
            "get": {
                "description": "get the status of server.",
                "consumes": [
                    "*/*"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "root"
                ],
                "summary": "Show the status of server.",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/v1/check-email": {
            "post": {
                "produces": [
                    "application/json"
                ],
                "summary": "Get attributes for the authenticated user. If multiple records for the same user, gets the one with the email confirmed.",
                "parameters": [
                    {
                        "description": "Specify the email to check.",
                        "name": "checkEmailRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.CheckEmailRequest"
                        }
                    }
                ],
                "responses": {
                    "0": {
                        "description": "",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.CheckEmailResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Get attributes for the authenticated user. If multiple records for the same user, gets the one with the email confirmed.",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.UserResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            },
            "put": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Modify attributes for the authenticated user",
                "parameters": [
                    {
                        "description": "New field values",
                        "name": "userUpdateRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.UserUpdateRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.UserResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            },
            "delete": {
                "summary": "Delete the authenticated user. Fails if the user has any devices.",
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "409": {
                        "description": "Returned if the user still has devices.",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user/agree-tos": {
            "post": {
                "summary": "Agree to the current terms of service",
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user/confirm-email": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "summary": "Submit an email confirmation key",
                "parameters": [
                    {
                        "description": "Specifies the key from the email",
                        "name": "confirmEmailRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ConfirmEmailRequest"
                        }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user/send-confirmation-email": {
            "post": {
                "summary": "Send a confirmation email to the authenticated user",
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user/set-migrated": {
            "post": {
                "summary": "Sets the migration timestamp.",
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user/submit-referral-code": {
            "post": {
                "summary": "Takes the referral code, validates and stores it",
                "parameters": [
                    {
                        "description": "ReferralCode is the 6-digit, alphanumeric referral code from another user.",
                        "name": "submitReferralCodeRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.SubmitReferralCodeRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.SubmitReferralCodeResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user/web3/challenge/generate": {
            "post": {
                "summary": "Generate a challenge message for the user to sign.",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ChallengeResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user/web3/challenge/submit": {
            "post": {
                "summary": "Confirm ownership of an ethereum address by submitting a signature",
                "parameters": [
                    {
                        "description": "Signed challenge message",
                        "name": "confirmEthereumRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ConfirmEthereumRequest"
                        }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v2/user": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Get attributes for the authenticated user. If multiple records for the same user, gets the one with the email confirmed.",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.UserResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controllers.ErrorResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "internal_controllers.ChallengeResponse": {
            "type": "object",
            "properties": {
                "challenge": {
                    "description": "Challenge is the message to be signed.",
                    "type": "string"
                },
                "expiresAt": {
                    "description": "ExpiresAt is the time at which the signed challenge will no longer be accepted.",
                    "type": "string"
                }
            }
        },
        "internal_controllers.CheckEmailRequest": {
            "type": "object",
            "properties": {
                "address": {
                    "description": "Address is the email address to check. Must be confirmed.",
                    "type": "string",
                    "example": "thaler@a16z.com"
                }
            }
        },
        "internal_controllers.CheckEmailResponse": {
            "type": "object",
            "properties": {
                "inUse": {
                    "description": "InUse specifies whether the email is attached to a DIMO user.",
                    "type": "boolean"
                },
                "wallets": {
                    "type": "object",
                    "properties": {
                        "external": {
                            "type": "integer"
                        },
                        "inApp": {
                            "type": "integer"
                        }
                    }
                }
            }
        },
        "internal_controllers.ConfirmEmailRequest": {
            "type": "object",
            "properties": {
                "key": {
                    "description": "Key is the 6-digit number from the confirmation email",
                    "type": "string",
                    "example": "010990"
                }
            }
        },
        "internal_controllers.ConfirmEthereumRequest": {
            "type": "object",
            "properties": {
                "signature": {
                    "description": "Signature is the result of signing the provided challenge message using the address in\nquestion.",
                    "type": "string"
                }
            }
        },
        "internal_controllers.ErrorResponse": {
            "type": "object",
            "properties": {
                "errorMessage": {
                    "type": "string"
                }
            }
        },
        "internal_controllers.SubmitReferralCodeRequest": {
            "type": "object",
            "properties": {
                "referralCode": {
                    "description": "ReferralCode is the 6-digit, alphanumeric referral code from another user.",
                    "type": "string",
                    "example": "ANB95N"
                }
            }
        },
        "internal_controllers.SubmitReferralCodeResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string"
                }
            }
        },
        "internal_controllers.UserResponse": {
            "type": "object",
            "properties": {
                "agreedTosAt": {
                    "description": "AgreedTosAt is the time at which the user last agreed to the terms of service.",
                    "type": "string",
                    "example": "2021-12-01T09:00:41Z"
                },
                "countryCode": {
                    "description": "CountryCode, if present, is a valid ISO 3166-1 alpha-3 country code.",
                    "type": "string",
                    "example": "USA"
                },
                "createdAt": {
                    "description": "CreatedAt is when the user first logged in.",
                    "type": "string",
                    "example": "2021-12-01T09:00:00Z"
                },
                "email": {
                    "description": "Email describes the user's email and the state of its confirmation.",
                    "allOf": [
                        {
                            "$ref": "#/definitions/internal_controllers.UserResponseEmail"
                        }
                    ]
                },
                "id": {
                    "description": "ID is the user's DIMO-internal ID.",
                    "type": "string",
                    "example": "ChFrb2JsaXR6QGRpbW8uem9uZRIGZ29vZ2xl"
                },
                "migratedAt": {
                    "type": "string",
                    "example": "2024-09-17T09:00:00Z"
                },
                "referralCode": {
                    "description": "ReferralCode is the user's referral code to be given to others. It is an 8 alphanumeric code,\nonly present if the account has a confirmed Ethereum address.",
                    "type": "string",
                    "example": "ANB95N"
                },
                "referredAt": {
                    "type": "string",
                    "example": "2021-12-01T09:00:41Z"
                },
                "referredBy": {
                    "type": "string",
                    "example": "0x3497B704a954789BC39999262510DE9B09Ff1366"
                },
                "web3": {
                    "description": "Web3 describes the user's blockchain account.",
                    "allOf": [
                        {
                            "$ref": "#/definitions/internal_controllers.UserResponseWeb3"
                        }
                    ]
                }
            }
        },
        "internal_controllers.UserResponseEmail": {
            "type": "object",
            "properties": {
                "address": {
                    "description": "Address is the email address for the user.",
                    "type": "string",
                    "example": "koblitz@dimo.zone"
                },
                "confirmationSentAt": {
                    "description": "ConfirmationSentAt is the time at which we last sent a confirmation email. This will only\nbe present if we've sent an email but the code has not been sent back to us.",
                    "type": "string",
                    "example": "2021-12-01T09:01:12Z"
                },
                "confirmed": {
                    "description": "Confirmed indicates whether the user has confirmed the address by entering a code.",
                    "type": "boolean",
                    "example": false
                }
            }
        },
        "internal_controllers.UserResponseWeb3": {
            "type": "object",
            "properties": {
                "address": {
                    "description": "Address is the Ethereum address associated with the user.",
                    "type": "string",
                    "example": "0x142e0C7A098622Ea98E5D67034251C4dFA746B5d"
                },
                "challengeSentAt": {
                    "description": "ChallengeSentAt is the time at which we last generated a challenge message for the user to\nsign. This will only be present if we've generated such a message but a signature has not\nbeen sent back to us.",
                    "type": "string",
                    "example": "2021-12-01T09:01:12Z"
                },
                "confirmed": {
                    "description": "Confirmed indicates whether the user has confirmed the address by signing a challenge\nmessage.",
                    "type": "boolean",
                    "example": false
                },
                "inApp": {
                    "description": "InApp indicates whether this is an in-app wallet, managed by the DIMO app.",
                    "type": "boolean",
                    "example": false
                },
                "used": {
                    "description": "Used indicates whether the user has used this address to perform any on-chain\nactions like minting, claiming, or pairing.",
                    "type": "boolean",
                    "example": false
                }
            }
        },
        "internal_controllers.UserUpdateRequest": {
            "type": "object",
            "properties": {
                "countryCode": {
                    "description": "CountryCode, if specified, should be a valid ISO 3166-1 alpha-3 country code",
                    "type": "string",
                    "example": "USA"
                },
                "email": {
                    "type": "object",
                    "properties": {
                        "address": {
                            "description": "Address, if present, should be a valid email address. Note when this field\nis modified the user's verification status will reset.",
                            "type": "string",
                            "example": "neal@dimo.zone"
                        }
                    }
                },
                "web3": {
                    "type": "object",
                    "properties": {
                        "address": {
                            "description": "Address, if present, should be a valid ethereum address. Note when this field\nis modified the user's address verification status will reset.",
                            "type": "string",
                            "example": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
                        },
                        "inApp": {
                            "description": "InApp, if true, indicates that the address above corresponds to an in-app wallet.\nYou can only set this when setting a new wallet. It defaults to false.",
                            "type": "boolean",
                            "example": true
                        }
                    }
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "DIMO User API",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
