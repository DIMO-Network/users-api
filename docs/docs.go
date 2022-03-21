// Package docs GENERATED BY THE COMMAND ABOVE; DO NOT EDIT
// This file was generated by swaggo/swag
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
        "/v1/user": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "summary": "Get attributes for the authenticated user",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/controllers.UserResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
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
                            "$ref": "#/definitions/controllers.UserUpdateRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/controllers.UserResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    }
                }
            },
            "delete": {
                "summary": "Delete the authenticated user",
                "responses": {
                    "204": {
                        "description": ""
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
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
                        "description": ""
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
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
                            "$ref": "#/definitions/controllers.ConfirmEmailRequest"
                        }
                    }
                ],
                "responses": {
                    "204": {
                        "description": ""
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user/generate-ethereum-challenge": {
            "post": {
                "summary": "Generate a challenge message for the user to sign.",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/controllers.ChallengeResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
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
                        "description": ""
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/v1/user/submit-ethereum-challenge": {
            "post": {
                "summary": "Confirm ownership of an ethereum address by submitting a signature",
                "parameters": [
                    {
                        "description": "Signed challenge message",
                        "name": "confirmEthereumRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/controllers.ConfirmEthereumRequest"
                        }
                    }
                ],
                "responses": {
                    "204": {
                        "description": ""
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/controllers.ErrorResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "controllers.ChallengeResponse": {
            "type": "object",
            "properties": {
                "challenge": {
                    "type": "string"
                },
                "expiresAt": {
                    "type": "string"
                }
            }
        },
        "controllers.ConfirmEmailRequest": {
            "type": "object",
            "properties": {
                "key": {
                    "description": "Key is the 6-digit number from the confirmation email",
                    "type": "string",
                    "example": "010990"
                }
            }
        },
        "controllers.ConfirmEthereumRequest": {
            "type": "object",
            "properties": {
                "signature": {
                    "type": "string"
                }
            }
        },
        "controllers.ErrorResponse": {
            "type": "object",
            "properties": {
                "errorMessage": {
                    "type": "string"
                }
            }
        },
        "controllers.UserResponse": {
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
                    "$ref": "#/definitions/controllers.UserResponseEmail"
                },
                "id": {
                    "description": "ID is the user's DIMO-internal ID.",
                    "type": "string",
                    "example": "ChFrb2JsaXR6QGRpbW8uem9uZRIGZ29vZ2xl"
                },
                "referralCode": {
                    "description": "ReferralCode is the short code used in a user's share link.",
                    "type": "string",
                    "example": "bUkZuSL7"
                },
                "referralsMade": {
                    "description": "ReferralsMade is the number of completed referrals made by the user",
                    "type": "integer",
                    "example": 1
                },
                "referredBy": {
                    "description": "ReferredBy is the referral code of the person who referred this user to the site.",
                    "type": "string",
                    "example": "k9H7RoTG"
                },
                "web3": {
                    "description": "Web3 describes the user's blockchain account.",
                    "$ref": "#/definitions/controllers.UserResponseWeb3"
                }
            }
        },
        "controllers.UserResponseEmail": {
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
        "controllers.UserResponseWeb3": {
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
                }
            }
        },
        "controllers.UserUpdateRequest": {
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
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
