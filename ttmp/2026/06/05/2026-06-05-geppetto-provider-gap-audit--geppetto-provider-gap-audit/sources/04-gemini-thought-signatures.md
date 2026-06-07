---
Title: Gemini thought signatures
SourceURL: https://ai.google.dev/gemini-api/docs/thought-signatures
SourceTool: defuddle
FetchedAt: 2026-06-05T08:08:32-04:00
Ticket: 2026-06-05-geppetto-provider-gap-audit
Topics:
  - geppetto
  - providers
  - api-docs
DocType: source
Summary: Official provider documentation captured for the Geppetto provider gap audit.
---

Las firmas de pensamiento son representaciones encriptadas del proceso de pensamiento interno del 
modelo y se usan para preservar el contexto de razonamiento en las interacciones de varios pasos. 
Cuando se usan modelos de pensamiento (como las series de Gemini 3 y 2.5), la API puede devolver un 
campo `thoughtSignature` dentro de las [partes de 
contenido](https://ai.google.dev/api/caching?hl=es-419#Part) de la respuesta (p.ej., partes `text` 
o `functionCall`).

Como regla general, si recibes una firma de pensamiento en la respuesta de un modelo, debes 
devolverla exactamente como la recibiste cuando envíes el historial de conversación en el 
siguiente turno. **Cuando uses los modelos de Gemini 3, debes devolver las firmas de pensamiento 
durante la llamada a funciones. De lo contrario, recibirás un error de validación** (código de 
estado 4xx). Esto incluye el uso del parámetro de configuración `minimal` [nivel de 
razonamiento](https://ai.google.dev/gemini-api/docs/thinking?hl=es-419#thinking-levels) para Gemini 
3 Flash.

## Cómo funciona

En el siguiente gráfico, se visualiza el significado de "turno" y "paso" en relación con la 
[llamada a función](https://ai.google.dev/gemini-api/docs/function-calling?hl=es-419) en la API de 
Gemini. Un "turno" es un intercambio único y completo en una conversación entre un usuario y un 
modelo. Un "paso" es una acción u operación más detallada que realiza el modelo, a menudo como 
parte de un proceso más grande para completar un turno.

![Diagrama de turnos y pasos de la llamada a 
función](https://ai.google.dev/static/gemini-api/docs/images/fc-turns.png?hl=es-419)

*Este documento se centra en el control de las llamadas a funciones para los modelos de Gemini 3. 
Consulta la sección sobre el [comportamiento del modelo](#model-behavior) para conocer las 
discrepancias con la versión 2.5.*

Gemini 3 devuelve firmas de pensamiento para todas las respuestas del modelo (respuestas de la API) 
con una llamada a función. Las firmas de pensamiento aparecen en los siguientes casos:

- Cuando hay llamadas a [funciones 
paralelas](https://ai.google.dev/gemini-api/docs/function-calling?hl=es-419#parallel_function_callin
g), la primera parte de la llamada a función que devuelve la respuesta del modelo tendrá una 
firma de pensamiento.
- Cuando hay llamadas a funciones secuenciales (de varios pasos), cada llamada a función tendrá 
una firma y debes pasar todas las firmas.
- Las respuestas del modelo sin una llamada a función devolverán una firma de pensamiento dentro 
de la última parte que muestre el modelo.

En la siguiente tabla, se proporciona una visualización de las llamadas a funciones de varios 
pasos, en la que se combinan las definiciones de turnos y pasos con el concepto de firmas que se 
presentó anteriormente:

| **Turn** | **Step** | **Solicitud del usuario** | **Respuesta del modelo** | **FunctionResponse** 
|
| --- | --- | --- | --- | --- |
| 1 | 1 | `request1 = user_prompt` | `FC1 + signature` | `FR1` |
| 1 | 2 | `request2 = request1 + (FC1 + signature) + FR1` | `FC2 + signature` | `FR2` |
| 1 | 3 | `request3 = request2 + (FC2 + signature) + FR2` | `text_output`  `(no FCs)` | Ninguno |

## Firmas en las partes de llamadas a funciones

Cuando Gemini genera un `functionCall`, se basa en el `thought_signature` para procesar 
correctamente el resultado de la herramienta en el siguiente turno.

- **Comportamiento**:
	- **Llamada a una sola función**: La parte `functionCall` contendrá un 
`thought_signature`.
		- **Llamadas a funciones paralelas**: Si el modelo genera llamadas a funciones 
paralelas en una respuesta, el `thought_signature` se adjunta **solo a la primera** parte de 
`functionCall`. Las partes `functionCall` posteriores de la misma respuesta **no** contendrán una 
firma.
- **Requisito**: **Debes** devolver esta firma en la parte exacta en la que se recibió cuando 
envíes el historial de conversación.
- **Validación**: Se aplica una validación estricta para todas las llamadas a funciones dentro 
del turno actual. (Solo se requiere el turno actual; no validamos los turnos anteriores)
	- La API retrocede en el historial (de más reciente a más antiguo) para encontrar el 
mensaje de **User** más reciente que contiene contenido estándar (p.ej., `text`) ( que sería el 
inicio del turno actual). No **be** un `functionResponse`.
		- Todas las respuestas del modelo **All** que se producen después de ese mensaje 
de uso específico se consideran parte de la respuesta.`functionCall`
		- **La primera parte `functionCall` de **cada paso** del turno actual **debe** 
incluir su `thought_signature`.**
		- Si omites un `thought_signature` para la primera parte de `functionCall` en 
cualquier paso del turno actual, la solicitud fallará con un error 400.
- **Si no se devuelven las firmas adecuadas, así se producirá el error**
	- Modelos de Gemini 3: Si no se incluyen firmas, se generará un error 400. La redacción 
tendrá el siguiente formato:
		- La llamada a función `<Function Call>` en el bloque de contenido `<index of 
contents array>` carece de un `thought_signature`. Por ejemplo, *Falta un `thought_signature` en la 
llamada a la función `FC1` en el bloque de contenido `1.`.*

### Ejemplo de llamada a función secuencial

En esta sección, se muestra un ejemplo de varias llamadas a funciones en el que el usuario hace 
una pregunta compleja que requiere varias tareas.

Veamos un ejemplo de llamada a función de varios turnos en el que el usuario hace una pregunta 
compleja que requiere varias tareas: `"Check flight status for AA100 and book a taxi if delayed"`.

| **Turn** | **Step** | **Solicitud del usuario** | **Respuesta del modelo** | **FunctionResponse** 
|
| --- | --- | --- | --- | --- |
| 1 | 1 | `request1="Check flight status for AA100 and book a taxi 2 hours before if delayed."` | 
`FC1 ("check_flight") + signature` | `FR1` |
| 1 | 2 | `request2 = request1 + FC1 ("check_flight") + signature + FR1` | `FC2("book_taxi") + 
signature` | `FR2` |
| 1 | 3 | `request3 = request2 + FC2 ("book_taxi") + signature + FR2` | `text_output`  `(no FCs)` | 
`None` |

El siguiente código ilustra la secuencia de la tabla anterior.

**Turno 1, paso 1 (solicitud del usuario)**

```
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "Check flight status for AA100 and book a taxi 2 hours before if delayed."
        }
      ]
    }
  ],
  "tools": [
    {
      "functionDeclarations": [
        {
          "name": "check_flight",
          "description": "Gets the current status of a flight",
          "parameters": {
            "type": "object",
            "properties": {
              "flight": {
                "type": "string",
                "description": "The flight number to check"
              }
            },
            "required": [
              "flight"
            ]
          }
        },
        {
          "name": "book_taxi",
          "description": "Book a taxi",
          "parameters": {
            "type": "object",
            "properties": {
              "time": {
                "type": "string",
                "description": "time to book the taxi"
              }
            },
            "required": [
              "time"
            ]
          }
        }
      ]
    }
  ]
}
```

**Turn 1, Step 1 (Model response)**

```
{
"content": {
        "role": "model",
        "parts": [
          {
            "functionCall": {
              "name": "check_flight",
              "args": {
                "flight": "AA100"
              }
            },
            "thoughtSignature": "<Signature A>"
          }
        ]
  }
}
```

**Turn 1, Step 2 (User response - Sending tool outputs)** Dado que este turno del usuario solo 
contiene un `functionResponse` (sin texto nuevo), seguimos en el turno 1. Debemos preservar 
`<Signature_A>`.

```
{
      "role": "user",
      "parts": [
        {
          "text": "Check flight status for AA100 and book a taxi 2 hours before if delayed."
        }
      ]
    },
    {
        "role": "model",
        "parts": [
          {
            "functionCall": {
              "name": "check_flight",
              "args": {
                "flight": "AA100"
              }
            },
            "thoughtSignature": "<Signature A>" //Required and Validated
          }
        ]
      },
      {
        "role": "user",
        "parts": [
          {
            "functionResponse": {
              "name": "check_flight",
              "response": {
                "status": "delayed",
                "departure_time": "12 PM"
                }
              }
            }
        ]
}
```

**Turno 1, paso 2 (modelo)** Ahora, el modelo decide reservar un taxi según el resultado de la 
herramienta anterior.

```
{
      "content": {
        "role": "model",
        "parts": [
          {
            "functionCall": {
              "name": "book_taxi",
              "args": {
                "time": "10 AM"
              }
            },
            "thoughtSignature": "<Signature B>"
          }
        ]
      }
}
```

**Turno 1, paso 3 (Usuario: Envío del resultado de la herramienta)** Para enviar la confirmación 
de la reserva de taxi, debemos incluir firmas para **TODAS** las llamadas a funciones en este bucle 
(`<Signature A>` + `<Signature B>`).

```
{
      "role": "user",
      "parts": [
        {
          "text": "Check flight status for AA100 and book a taxi 2 hours before if delayed."
        }
      ]
    },
    {
        "role": "model",
        "parts": [
          {
            "functionCall": {
              "name": "check_flight",
              "args": {
                "flight": "AA100"
              }
            },
            "thoughtSignature": "<Signature A>" //Required and Validated
          }
        ]
      },
      {
        "role": "user",
        "parts": [
          {
            "functionResponse": {
              "name": "check_flight",
              "response": {
                "status": "delayed",
                "departure_time": "12 PM"
              }
              }
            }
        ]
      },
      {
        "role": "model",
        "parts": [
          {
            "functionCall": {
              "name": "book_taxi",
              "args": {
                "time": "10 AM"
              }
            },
            "thoughtSignature": "<Signature B>" //Required and Validated
          }
        ]
      },
      {
        "role": "user",
        "parts": [
          {
            "functionResponse": {
              "name": "book_taxi",
              "response": {
                "booking_status": "success"
              }
              }
            }
        ]
    }
}
```

### Ejemplo de llamada a función paralela

Analicemos un ejemplo de llamada a función paralela en el que los usuarios le piden a `"Check 
weather in Paris and London"` que muestre dónde el modelo realiza la validación.

| **Turn** | **Step** | **Solicitud del usuario** | **Respuesta del modelo** | **FunctionResponse** 
|
| --- | --- | --- | --- | --- |
| 1 | 1 | `request1="Check the weather in Paris and London"` | FC1 ("París") + firma  FC2 
("Londres") | FR1 |
| 1 | 2 | `request 2 = request1 + FC1 ("Paris") + signature + FC2 ("London")` | text\_output  (sin 
FC) | Ninguno |

El siguiente código ilustra la secuencia de la tabla anterior.

**Turno 1, paso 1 (solicitud del usuario)**

```
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "Check the weather in Paris and London."
        }
      ]
    }
  ],
  "tools": [
    {
      "functionDeclarations": [
        {
          "name": "get_current_temperature",
          "description": "Gets the current temperature for a given location.",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {
                "type": "string",
                "description": "The city name, e.g. San Francisco"
              }
            },
            "required": [
              "location"
            ]
          }
        }
      ]
    }
  ]
}
```

**Turn 1, Step 1 (Model response)**

```
{
  "content": {
    "parts": [
      {
        "functionCall": {
          "name": "get_current_temperature",
          "args": {
            "location": "Paris"
          }
        },
        "thoughtSignature": "<Signature_A>"// INCLUDED on First FC
      },
      {
        "functionCall": {
          "name": "get_current_temperature",
          "args": {
            "location": "London"
          }// NO signature on subsequent parallel FCs
        }
      }
    ]
  }
}
```

**Turn 1, Step 2 (User response - Sending tool outputs)** Debemos conservar `<Signature_A>` en la 
primera parte exactamente como se recibió.

```
[
  {
    "role": "user",
    "parts": [
      {
        "text": "Check the weather in Paris and London."
      }
    ]
  },
  {
    "role": "model",
    "parts": [
      {
        "functionCall": {
          "name": "get_current_temperature",
          "args": {
            "city": "Paris"
          }
        },
        "thought_signature": "<Signature_A>" // MUST BE INCLUDED
      },
      {
        "functionCall": {
          "name": "get_current_temperature",
          "args": {
            "city": "London"
          }
        }
      } // NO SIGNATURE FIELD
    ]
  },
  {
    "role": "user",
    "parts": [
      {
        "functionResponse": {
          "name": "get_current_temperature",
          "response": {
            "temp": "15C"
          }
        }
      },
      {
        "functionResponse": {
          "name": "get_current_temperature",
          "response": {
            "temp": "12C"
          }
        }
      }
    ]
  }
]
```

## Firmas en partes que no son de functionCall

Gemini también puede devolver `thought_signatures` en la parte final de la respuesta en las partes 
que no son de llamadas a funciones.

- **Comportamiento**: La parte final del contenido (`text, inlineData…`) que devuelve el modelo 
puede contener un `thought_signature`.
- **Recomendación**: Se **recomienda** devolver estas firmas para garantizar que el modelo 
mantenga un razonamiento de alta calidad, especialmente para el seguimiento de instrucciones 
complejas o flujos de trabajo de agentes simulados.
- **Validación**: La API **no** aplica la validación de forma estricta. No recibirás un error de 
bloqueo si los omites, aunque el rendimiento puede disminuir.

### Texto/Razonamiento en contexto (sin validación)

**Turn 1, Step 1 (Model response)**

```
{
  "role": "model",
  "parts": [
    {
      "text": "I need to calculate the risk. Let me think step-by-step...",
      "thought_signature": "<Signature_C>" // OPTIONAL (Recommended)
    }
  ]
}
```

**Turn 2, Step 1 (User)**

```
[
  { "role": "user", "parts": [{ "text": "What is the risk?" }] },
  {
    "role": "model", 
    "parts": [
      {
        "text": "I need to calculate the risk. Let me think step-by-step...",
        // If you omit <Signature_C> here, no error will occur.
      }
    ]
  },
  { "role": "user", "parts": [{ "text": "Summarize it." }] }
]
```

## Conservación del pensamiento y uso de tokens

**A partir de Gemini 3.5 Flash**, el modelo usa el contexto de razonamiento de todos los turnos 
anteriores cuando hay firmas de pensamiento en el historial de conversación.

Para habilitar la conservación del pensamiento, **pasa el historial de conversación completo y 
sin modificar** (incluidos los campos `thought_signature` que se devolvieron en los turnos 
anteriores del modelo) en el array `contents` de tu solicitud.

### Administra el consumo de tokens

Conservar los pensamientos intermedios en varios turnos aumenta el recuento de tokens de entrada en 
los turnos posteriores, ya que el modelo debe analizar las firmas de pensamiento de los turnos 
anteriores.

Si tu aplicación realiza consultas simples o deseas minimizar los costos en conversaciones largas, 
puedes borrar las firmas de pensamiento anteriores del historial de conversación.

## Firmas para la compatibilidad con OpenAI

En el siguiente ejemplo, se muestra cómo controlar las firmas de pensamiento para una API de 
finalización de chat con [compatibilidad con 
OpenAI](https://ai.google.dev/gemini-api/docs/openai?hl=es-419).

### Ejemplo de llamada a función secuencial

Este es un ejemplo de llamada a varias funciones en el que el usuario hace una pregunta compleja 
que requiere varias tareas.

Veamos un ejemplo de llamada a función de varios turnos en el que el usuario pregunta `Check 
flight status for AA100 and book a taxi if delayed` y puedes ver qué sucede cuando el usuario hace 
una pregunta compleja que requiere varias tareas.

| **Turn** | **Step** | **Solicitud del usuario** | **Respuesta del modelo** | **FunctionResponse** 
|
| --- | --- | --- | --- | --- |
| 1 | 1 | `request1 = "Check flight status for AA100 and book a taxi 2 hours before if delayed."` | 
`FC1 ("check_flight") + signature` | `FR1` |
| 1 | 2 | `request2 = request1 + FC1 ("check_flight") + signature + FR1` | `FC2("book_taxi") + 
signature` | `FR2` |
| 1 | 3 | `request3 = request2 + FC2 ("book_taxi") + signature + FR2` | `text_output`  `(no FCs)` | 
`None` |

El siguiente código recorre la secuencia determinada.

**Turno 1, paso 1 (solicitud del usuario)**

```
{
  "model": "google/gemini-3.1-pro-preview",
  "messages": [
    {
      "role": "user",
      "content": "Check flight status for AA100 and book a taxi 2 hours before if delayed."
    }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "check_flight",
        "description": "Gets the current status of a flight",
        "parameters": {
          "type": "object",
          "properties": {
            "flight": {
              "type": "string",
              "description": "The flight number to check."
            }
          },
          "required": [
            "flight"
          ]
        }
      }
    },
    {
      "type": "function",
      "function": {
        "name": "book_taxi",
        "description": "Book a taxi",
        "parameters": {
          "type": "object",
          "properties": {
            "time": {
              "type": "string",
              "description": "time to book the taxi"
            }
          },
          "required": [
            "time"
          ]
        }
      }
    }
  ]
}
```

**Turno 1, paso 1 (respuesta del modelo)**

```
{
      "role": "model",
        "tool_calls": [
          {
            "extra_content": {
              "google": {
                "thought_signature": "<Signature A>"
              }
            },
            "function": {
              "arguments": "{\"flight\":\"AA100\"}",
              "name": "check_flight"
            },
            "id": "function-call-1",
            "type": "function"
          }
        ]
    }
```

**Turno 1, paso 2 (respuesta del usuario: envío de resultados de herramientas)**

Como este turno del usuario solo contiene un `functionResponse` (sin texto nuevo), seguimos en el 
turno 1 y debemos conservar `<Signature_A>`.

```
"messages": [
    {
      "role": "user",
      "content": "Check flight status for AA100 and book a taxi 2 hours before if delayed."
    },
    {
      "role": "model",
        "tool_calls": [
          {
            "extra_content": {
              "google": {
                "thought_signature": "<Signature A>" //Required and Validated
              }
            },
            "function": {
              "arguments": "{\"flight\":\"AA100\"}",
              "name": "check_flight"
            },
            "id": "function-call-1",
            "type": "function"
          }
        ]
    },
    {
      "role": "tool",
      "name": "check_flight",
      "tool_call_id": "function-call-1",
      "content": "{\"status\":\"delayed\",\"departure_time\":\"12 PM\"}"                 
    }
  ]
```

**Turn 1, Step 2 (Model)**

Ahora, el modelo decide reservar un taxi según el resultado anterior de la herramienta.

```
{
"role": "model",
"tool_calls": [
{
"extra_content": {
"google": {
"thought_signature": "<Signature B>"
}
            },
            "function": {
              "arguments": "{\"time\":\"10 AM\"}",
              "name": "book_taxi"
            },
            "id": "function-call-2",
            "type": "function"
          }
       ]
}
```

**Turno 1, paso 3 (usuario: envío del resultado de la herramienta)**

Para enviar la confirmación de la reserva de taxi, debemos incluir firmas para TODAS las llamadas 
a funciones en este bucle (`<Signature A>` + `<Signature B>`).

```
"messages": [
    {
      "role": "user",
      "content": "Check flight status for AA100 and book a taxi 2 hours before if delayed."
    },
    {
      "role": "model",
        "tool_calls": [
          {
            "extra_content": {
              "google": {
                "thought_signature": "<Signature A>" //Required and Validated
              }
            },
            "function": {
              "arguments": "{\"flight\":\"AA100\"}",
              "name": "check_flight"
            },
            "id": "function-call-1d6a1a61-6f4f-4029-80ce-61586bd86da5",
            "type": "function"
          }
        ]
    },
    {
      "role": "tool",
      "name": "check_flight",
      "tool_call_id": "function-call-1d6a1a61-6f4f-4029-80ce-61586bd86da5",
      "content": "{\"status\":\"delayed\",\"departure_time\":\"12 PM\"}"                 
    },
    {
      "role": "model",
        "tool_calls": [
          {
            "extra_content": {
              "google": {
                "thought_signature": "<Signature B>" //Required and Validated
              }
            },
            "function": {
              "arguments": "{\"time\":\"10 AM\"}",
              "name": "book_taxi"
            },
            "id": "function-call-65b325ba-9b40-4003-9535-8c7137b35634",
            "type": "function"
          }
        ]
    },
    {
      "role": "tool",
      "name": "book_taxi",
      "tool_call_id": "function-call-65b325ba-9b40-4003-9535-8c7137b35634",
      "content": "{\"booking_status\":\"success\"}"
    }
  ]
```

### Ejemplo de llamada a función paralela

Analicemos un ejemplo de llamada a función paralela en el que el usuario pregunta `"Check weather 
in Paris and London"` y puedes ver dónde el modelo realiza la validación.

| **Turn** | **Step** | **Solicitud del usuario** | **Respuesta del modelo** | **FunctionResponse** 
|
| --- | --- | --- | --- | --- |
| 1 | 1 | `request1="Check the weather in Paris and London"` | `FC1 ("Paris") + signature`  `FC2 
("London")` | `FR1` |
| 1 | 2 | `request 2 = request1 + FC1 ("Paris") + signature + FC2 ("London")` | `text_output`  `(no 
FCs)` | `None` |

Este es el código para recorrer la secuencia determinada.

**Turno 1, paso 1 (solicitud del usuario)**

```
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "Check the weather in Paris and London."
        }
      ]
    }
  ],
  "tools": [
    {
      "functionDeclarations": [
        {
          "name": "get_current_temperature",
          "description": "Gets the current temperature for a given location.",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {
                "type": "string",
                "description": "The city name, e.g. San Francisco"
              }
            },
            "required": [
              "location"
            ]
          }
        }
      ]
    }
  ]
}
```

**Turno 1, paso 1 (respuesta del modelo)**

```
{
"role": "assistant",
        "tool_calls": [
          {
            "extra_content": {
              "google": {
                "thought_signature": "<Signature A>" //Signature returned
              }
            },
            "function": {
              "arguments": "{\"location\":\"Paris\"}",
              "name": "get_current_temperature"
            },
            "id": "function-call-f3b9ecb3-d55f-4076-98c8-b13e9d1c0e01",
            "type": "function"
          },
          {
            "function": {
              "arguments": "{\"location\":\"London\"}",
              "name": "get_current_temperature"
            },
            "id": "function-call-335673ad-913e-42d1-bbf5-387c8ab80f44",
            "type": "function" // No signature on Parallel FC
          }
        ]
}
```

**Turno 1, paso 2 (respuesta del usuario: envío de resultados de herramientas)**

Debes conservar `<Signature_A>` en la primera parte exactamente como la recibiste.

```
"messages": [
    {
      "role": "user",
      "content": "Check the weather in Paris and London."
    },
    {
      "role": "assistant",
        "tool_calls": [
          {
            "extra_content": {
              "google": {
                "thought_signature": "<Signature A>" //Required
              }
            },
            "function": {
              "arguments": "{\"location\":\"Paris\"}",
              "name": "get_current_temperature"
            },
            "id": "function-call-f3b9ecb3-d55f-4076-98c8-b13e9d1c0e01",
            "type": "function"
          },
          {
            "function": { //No Signature
              "arguments": "{\"location\":\"London\"}",
              "name": "get_current_temperature"
            },
            "id": "function-call-335673ad-913e-42d1-bbf5-387c8ab80f44",
            "type": "function"
          }
        ]
    },
    {
      "role":"tool",
      "name": "get_current_temperature",
      "tool_call_id": "function-call-f3b9ecb3-d55f-4076-98c8-b13e9d1c0e01",
      "content": "{\"temp\":\"15C\"}"
    },    
    {
      "role":"tool",
      "name": "get_current_temperature",
      "tool_call_id": "function-call-335673ad-913e-42d1-bbf5-387c8ab80f44",
      "content": "{\"temp\":\"12C\"}"
    }
  ]
```

## Preguntas frecuentes

1. **¿Cómo transfiero el historial de un modelo diferente a Gemini 3 con una parte de llamada a 
función en el turno y el paso actuales? ¿Necesito proporcionar partes de llamadas a funciones que 
no generó la API y, por lo tanto, no tienen una firma de pensamiento asociada?**
	Si bien se recomienda no insertar bloques de llamadas a funciones personalizadas en la 
solicitud, en los casos en que no se pueda evitar, p.ej., proporcionar información al modelo sobre 
las llamadas a funciones y las respuestas que el cliente ejecutó de forma determinística, o 
transferir un registro de otro modelo que no incluya firmas de pensamiento, puedes establecer las 
siguientes firmas ficticias de `"context_engineering_is_the_way_to_go"` o 
`"skip_thought_signature_validator"` en el campo de firma de pensamiento para omitir la validación.
2. **Envío respuestas y llamadas a funciones paralelas intercaladas, y la API devuelve un error 
400. ¿Por qué?**
	Cuando la API devuelve llamadas a funciones paralelas "FC1 + firma, FC2", la respuesta del 
usuario esperada es "FC1 + firma, FC2, FR1, FR2". Si los tienes intercalados como "FC1 + firma, 
FR1, FC2, FR2", la API devolverá un error 400.
3. **Cuando se transmite y el modelo no devuelve una llamada a función, no puedo encontrar la 
firma de pensamiento**
	Durante una respuesta del modelo que no contiene una FC con una solicitud de transmisión, 
el modelo puede devolver la firma de pensamiento en una parte con una parte de contenido de texto 
vacía. Se recomienda analizar toda la solicitud hasta que el modelo devuelva `finish_reason`.

## Firmas de pensamiento para diferentes modelos

Los [modelos de Gemini 3](https://ai.google.dev/gemini-api/docs/models?hl=es-419#gemini-3) y los 
modelos de Gemini 2.5 se comportan de manera diferente con las firmas de pensamiento:

- **Preservación del pensamiento**:
	- **A partir de Gemini 3.5 Flash**, el modelo usa el contexto de razonamiento de todos los 
turnos anteriores cuando hay firmas de pensamiento en el historial de conversación.
		- Los modelos anteriores no usan el contexto de razonamiento de los turnos 
anteriores de la misma manera.
- **Si hay llamadas a funciones en una respuesta**:
	- Gemini 3 siempre tendrá la firma en la primera parte de la llamada a función. Es 
**obligatorio** devolver esa parte.
		- Gemini 2.5 tendrá la firma en la primera parte (independientemente del tipo). Es 
**opcional** devolver esa parte.
- **Si no hay llamadas a funciones en una respuesta**, haz lo siguiente:
	- Gemini 3 tendrá la firma en la última parte si el modelo genera un pensamiento.
		- Gemini 2.5 no tendrá firma en ninguna parte.

Consulta la página [Thinking](https://ai.google.dev/gemini-api/docs/thinking?hl=es-419#signatures) 
para obtener más detalles de la comparación. Para los modelos de imágenes de Gemini 3, consulta 
la sección sobre el proceso de razonamiento de la guía de [generación de 
imágenes](https://ai.google.dev/gemini-api/docs/image-generation?hl=es-419#thinking-process).
