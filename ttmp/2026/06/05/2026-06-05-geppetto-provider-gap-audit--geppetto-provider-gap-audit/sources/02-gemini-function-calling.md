---
Title: Gemini function calling
SourceURL: https://ai.google.dev/gemini-api/docs/function-calling
SourceTool: defuddle
FetchedAt: 2026-06-05T08:08:27-04:00
Ticket: 2026-06-05-geppetto-provider-gap-audit
Topics:
  - geppetto
  - providers
  - api-docs
DocType: source
Summary: Official provider documentation captured for the Geppetto provider gap audit.
---

Com a chamada de função, é possível conectar modelos a ferramentas e APIs externas. Em vez de 
gerar respostas de texto, o modelo determina quando chamar funções específicas e fornece os 
parâmetros necessários para executar ações no mundo real. Isso permite que o modelo atue como 
uma ponte entre a linguagem natural e as ações e dados do mundo real. A chamada de função tem 
três casos de uso principais:

- **Aumentar o conhecimento**:acesse informações de fontes externas, como bancos de dados, APIs e 
bases de conhecimento.
- **Ampliar recursos**:use ferramentas externas para realizar cálculos e ampliar as limitações 
do modelo, como usar uma calculadora ou criar gráficos.
- **Realizar ações**:interaja com sistemas externos usando APIs, como agendar compromissos, criar 
faturas, enviar e-mails ou controlar dispositivos domésticos inteligentes.

### Python

```
from google import genai
from google.genai import types

# Define the function declaration for the model
schedule_meeting_function = {
    "name": "schedule_meeting",
    "description": "Schedules a meeting with specified attendees at a given time and date.",
    "parameters": {
        "type": "object",
        "properties": {
            "attendees": {
                "type": "array",
                "items": {"type": "string"},
                "description": "List of people attending the meeting.",
            },
            "date": {
                "type": "string",
                "description": "Date of the meeting (e.g., '2024-07-29')",
            },
            "time": {
                "type": "string",
                "description": "Time of the meeting (e.g., '15:00')",
            },
            "topic": {
                "type": "string",
                "description": "The subject or topic of the meeting.",
            },
        },
        "required": ["attendees", "date", "time", "topic"],
    },
}

# Configure the client and tools
client = genai.Client()
tools = types.Tool(function_declarations=[schedule_meeting_function])
config = types.GenerateContentConfig(tools=[tools])

# Send request with function declarations
response = client.models.generate_content(
    model="gemini-3.5-flash",
    contents="Schedule a meeting with Bob and Alice for 03/14/2025 at 10:00 AM about the Q3 
planning.",
    config=config,
)

# Check for a function call
if response.candidates[0].content.parts[0].function_call:
    function_call = response.candidates[0].content.parts[0].function_call
    print(f"Function to call: {function_call.name}")
    print(f"ID: {function_call.id}")
    print(f"Arguments: {function_call.args}")
    #  In a real app, you would call your function here:
    #  result = schedule_meeting(**function_call.args)
else:
    print("No function call found in the response.")
    print(response.text)
```

### JavaScript

```
import { GoogleGenAI, Type } from '@google/genai';

// Configure the client
const ai = new GoogleGenAI({});

// Define the function declaration for the model
const scheduleMeetingFunctionDeclaration = {
  name: 'schedule_meeting',
  description: 'Schedules a meeting with specified attendees at a given time and date.',
  parameters: {
    type: Type.OBJECT,
    properties: {
      attendees: {
        type: Type.ARRAY,
        items: { type: Type.STRING },
        description: 'List of people attending the meeting.',
      },
      date: {
        type: Type.STRING,
        description: 'Date of the meeting (e.g., "2024-07-29")',
      },
      time: {
        type: Type.STRING,
        description: 'Time of the meeting (e.g., "15:00")',
      },
      topic: {
        type: Type.STRING,
        description: 'The subject or topic of the meeting.',
      },
    },
    required: ['attendees', 'date', 'time', 'topic'],
  },
};

// Send request with function declarations
const response = await ai.models.generateContent({
  model: 'gemini-3.5-flash',
  contents: 'Schedule a meeting with Bob and Alice for 03/27/2025 at 10:00 AM about the Q3 
planning.',
  config: {
    tools: [{
      functionDeclarations: [scheduleMeetingFunctionDeclaration]
    }],
  },
});

// Check for function calls in the response
if (response.functionCalls && response.functionCalls.length > 0) {
  const functionCall = response.functionCalls[0]; // Assuming one function call
  console.log(\`Function to call: ${functionCall.name}\`);
  console.log(\`ID: ${functionCall.id}\`);
  console.log(\`Arguments: ${JSON.stringify(functionCall.args)}\`);
  // In a real app, you would call your actual function here:
  // const result = await scheduleMeeting(functionCall.args);
} else {
  console.log("No function call found in the response.");
  console.log(response.text);
}
```

### REST

```
curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash:generateContent" \
  -H "x-goog-api-key: $GEMINI_API_KEY" \
  -H 'Content-Type: application/json' \
  -X POST \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          {
            "text": "Schedule a meeting with Bob and Alice for 03/27/2025 at 10:00 AM about the Q3 
planning."
          }
        ]
      }
    ],
    "tools": [
      {
        "functionDeclarations": [
          {
            "name": "schedule_meeting",
            "description": "Schedules a meeting with specified attendees at a given time and date.",
            "parameters": {
              "type": "object",
              "properties": {
                "attendees": {
                  "type": "array",
                  "items": {"type": "string"},
                  "description": "List of people attending the meeting."
                },
                "date": {
                  "type": "string",
                  "description": "Date of the meeting (e.g., '2024-07-29')"
                },
                "time": {
                  "type": "string",
                  "description": "Time of the meeting (e.g., '15:00')"
                },
                "topic": {
                  "type": "string",
                  "description": "The subject or topic of the meeting."
                }
              },
              "required": ["attendees", "date", "time", "topic"]
            }
          }
        ]
      }
    ]
  }'
```

## Como a chamada de funções funciona

![Visão geral das chamadas de 
função](https://ai.google.dev/static/gemini-api/docs/images/function-calling-overview.png?hl=pt-br
)

A chamada de função envolve uma interação estruturada entre seu aplicativo, o modelo e 
funções externas. Confira os detalhes do processo:

1. **Definir declaração de função**:defina a declaração de função no código do aplicativo. 
As declarações de função descrevem o nome, os parâmetros e a finalidade da função para o 
modelo.
2. **Chamar a API com declarações de função**:envie o comando do usuário com as declarações 
de função para o modelo. Ele analisa a solicitação e determina se uma chamada de função seria 
útil. Se for o caso, ele responde com um objeto JSON estruturado que contém o nome da função, 
os argumentos e um `id` exclusivo. Esse `id` agora é sempre retornado pela API para modelos do 
Gemini 3 <sup>*</sup>.
3. **Executar o código da função (sua responsabilidade)**: o modelo *não* executa a função em 
si. É responsabilidade do aplicativo processar a resposta e verificar uma chamada de função. Se
	- **Sim**: extraia o nome, os argumentos e `id` da função e execute a função 
correspondente no seu aplicativo.
		- **Não**:o modelo forneceu uma resposta de texto direta ao comando (esse fluxo é 
menos enfatizado no exemplo, mas é um resultado possível).
4. **Crie uma resposta fácil de usar**:se uma função foi executada, capture o resultado e envie 
de volta ao modelo, incluindo o `id` correspondente em uma próxima vez da conversa. Ele vai usar o 
resultado para gerar uma resposta final e fácil de usar que incorpora as informações da chamada 
de função.

Esse processo pode ser repetido várias vezes, permitindo interações e fluxos de trabalho 
complexos. O modelo também permite chamar várias funções em um único turno ([chamada de 
função paralela](#parallel_function_calling)), em sequência ([chamada de função 
composicional](#compositional_function_calling)) e com ferramentas integradas do Gemini ([uso de 
várias ferramentas](#native-tools)).

<sup>*</sup> **Sempre mapear IDs de função**:agora o Gemini 3 sempre retorna um `id` exclusivo 
com cada `functionCall`. Inclua exatamente `id` no seu `functionResponse` para que o modelo possa 
mapear com precisão o resultado de volta à solicitação original.

### Etapa 1: definir uma declaração de função

Defina uma função e a declaração dela no código do aplicativo para que os usuários possam 
definir valores de luz e fazer uma solicitação de API. Essa função pode chamar serviços ou 
APIs externos.

### Python

```
# Define a function that the model can call to control smart lights
set_light_values_declaration = {
    "name": "set_light_values",
    "description": "Sets the brightness and color temperature of a light.",
    "parameters": {
        "type": "object",
        "properties": {
            "brightness": {
                "type": "integer",
                "description": "Light level from 0 to 100. Zero is off and 100 is full brightness",
            },
            "color_temp": {
                "type": "string",
                "enum": ["daylight", "cool", "warm"],
                "description": "Color temperature of the light fixture, which can be \`daylight\`, 
\`cool\` or \`warm\`.",
            },
        },
        "required": ["brightness", "color_temp"],
    },
}

# This is the actual function that would be called based on the model's suggestion
def set_light_values(brightness: int, color_temp: str) -> dict[str, int | str]:
    """Set the brightness and color temperature of a room light. (mock API).

    Args:
        brightness: Light level from 0 to 100. Zero is off and 100 is full brightness
        color_temp: Color temperature of the light fixture, which can be \`daylight\`, \`cool\` or 
\`warm\`.

    Returns:
        A dictionary containing the set brightness and color temperature.
    """
    return {"brightness": brightness, "colorTemperature": color_temp}
```

### JavaScript

```
import { Type } from '@google/genai';

// Define a function that the model can call to control smart lights
const setLightValuesFunctionDeclaration = {
  name: 'set_light_values',
  description: 'Sets the brightness and color temperature of a light.',
  parameters: {
    type: Type.OBJECT,
    properties: {
      brightness: {
        type: Type.NUMBER,
        description: 'Light level from 0 to 100. Zero is off and 100 is full brightness',
      },
      color_temp: {
        type: Type.STRING,
        enum: ['daylight', 'cool', 'warm'],
        description: 'Color temperature of the light fixture, which can be \`daylight\`, \`cool\` 
or \`warm\`.',
      },
    },
    required: ['brightness', 'color_temp'],
  },
};

/**

*   Set the brightness and color temperature of a room light. (mock API)
*   @param {number} brightness - Light level from 0 to 100. Zero is off and 100 is full brightness
*   @param {string} color_temp - Color temperature of the light fixture, which can be \`daylight\`, 
\`cool\` or \`warm\`.
*   @return {Object} A dictionary containing the set brightness and color temperature.
*/
function setLightValues(brightness, color_temp) {
  return {
    brightness: brightness,
    colorTemperature: color_temp
  };
}
```

### Etapa 2: chamar o modelo com declarações de função

Depois de definir as declarações de função, você pode pedir ao modelo para usá-las. Ele 
analisa o comando e as declarações de função e decide se vai responder diretamente ou chamar 
uma função. Se uma função for chamada, o objeto de resposta vai conter uma sugestão de chamada 
de função.

### Python

```
from google.genai import types

# Configure the client and tools
client = genai.Client()
tools = types.Tool(function_declarations=[set_light_values_declaration])
config = types.GenerateContentConfig(tools=[tools])

# Define user prompt
contents = [
    types.Content(
        role="user", parts=[types.Part(text="Turn the lights down to a romantic level")]
    )
]

# Send request with function declarations
response = client.models.generate_content(
    model="gemini-3.5-flash",
    contents=contents,
    config=config,
)

print(response.candidates[0].content.parts[0].function_call)
```

### JavaScript

```
import { GoogleGenAI } from '@google/genai';

// Generation config with function declaration
const config = {
  tools: [{
    functionDeclarations: [setLightValuesFunctionDeclaration]
  }]
};

// Configure the client
const ai = new GoogleGenAI({});

// Define user prompt
const contents = [
  {
    role: 'user',
    parts: [{ text: 'Turn the lights down to a romantic level' }]
  }
];

// Send request with function declarations
const response = await ai.models.generateContent({
  model: 'gemini-3.5-flash',
  contents: contents,
  config: config
});

console.log(response.functionCalls[0]);
```

Em seguida, o modelo retorna um objeto `functionCall` em um esquema compatível com OpenAPI que 
especifica como chamar uma ou mais das funções declaradas para responder à pergunta do usuário.

### Python

```
id='8f2b1a3c' args={'color_temp': 'warm', 'brightness': 25} name='set_light_values'
```

### JavaScript

```
{
  id: '8f2b1a3c',
  name: 'set_light_values',
  args: { brightness: 25, color_temp: 'warm' }
}
```

### Etapa 3: executar o código da função set\_light\_values

Extraia os detalhes da chamada de função da resposta do modelo, analise os argumentos e execute a 
função `set_light_values`.

### Python

```
# Extract tool call details, it may not be in the first part.
tool_call = response.candidates[0].content.parts[0].function_call

if tool_call.name == "set_light_values":
    result = set_light_values(**tool_call.args)
    print(f"Function execution result: {result}")
```

### JavaScript

```
// Extract tool call details
const tool_call = response.functionCalls[0]

let result;
if (tool_call.name === 'set_light_values') {
  result = setLightValues(tool_call.args.brightness, tool_call.args.color_temp);
  console.log(\`Function execution result: ${JSON.stringify(result)}\`);
}
```

### Etapa 4: criar uma resposta fácil de usar com o resultado da função e chamar o modelo 
novamente

Por fim, envie o resultado da execução da função de volta ao modelo para que ele possa 
incorporar essas informações na resposta final ao usuário.

### Python

```
from google import genai
from google.genai import types

# Create a function response part
function_response_part = types.Part.from_function_response(
    name=tool_call.name,
    response={"result": result},
    id=tool_call.id,
)

# Append function call and result of the function execution to contents
contents.append(response.candidates[0].content) # Append the content from the model's response.
contents.append(types.Content(role="user", parts=[function_response_part])) # Append the function 
response

client = genai.Client()
final_response = client.models.generate_content(
    model="gemini-3.5-flash",
    config=config,
    contents=contents,
)

print(final_response.text)
```

### JavaScript

```
// Create a function response part
const function_response_part = {
  name: tool_call.name,
  response: { result },
  id: tool_call.id
}

// Append function call and result of the function execution to contents
contents.push(response.candidates[0].content);
contents.push({ role: 'user', parts: [{ functionResponse: function_response_part }] });

// Get the final response from the model
const final_response = await ai.models.generateContent({
  model: 'gemini-3.5-flash',
  contents: contents,
  config: config
});

console.log(final_response.text);
```

Isso conclui o fluxo de chamadas de função. O modelo usou a função `set_light_values` para 
realizar a ação solicitada pelo usuário.

## Declarações de função

Ao implementar a chamada de função em um comando, você cria um objeto `tools`, que contém um ou 
mais `function declarations`. Você define funções usando JSON, especificamente com um 
[subconjunto selecionado](https://ai.google.dev/api/caching?hl=pt-br#Schema) do formato [esquema 
OpenAPI](https://spec.openapis.org/oas/v3.0.3#schemaw). Uma única declaração de função pode 
incluir os seguintes parâmetros:

- `name` (string): um nome exclusivo para a função (`get_weather_forecast`, `send_email`). Use 
nomes descritivos sem espaços ou caracteres especiais (use sublinhados ou camelCase).
- `description` (string): uma explicação clara e detalhada da finalidade e das capacidades da 
função. Isso é crucial para que o modelo entenda quando usar a função. Seja específico e dê 
exemplos, se necessário ("Encontra cinemas com base na localização e, opcionalmente, no título 
do filme que está em cartaz").
- `parameters` (objeto): define os parâmetros de entrada que a função espera.
	- `type` (string): especifica o tipo de dados geral, como `object`.
		- `properties` (objeto): lista parâmetros individuais, cada um com:
		- `type` (string): o tipo de dados do parâmetro, como `string`, `integer` e 
`boolean, array`.
				- `description` (string): uma descrição da finalidade e do 
formato do parâmetro. Forneça exemplos e restrições ("A cidade e o estado, por exemplo, 'São 
Francisco, CA' ou um CEP, por exemplo, '95616'").
				- `enum` (matriz, opcional): se os valores de parâmetro forem de 
um conjunto fixo, use "enum" para listar os valores permitidos em vez de apenas descrevê-los na 
descrição. Isso melhora a precisão ("enum": \["daylight", "cool", "warm"\]).
		- `required` (matriz): uma matriz de strings que lista os nomes dos parâmetros 
obrigatórios para o funcionamento da função.

Também é possível criar `FunctionDeclarations` diretamente de funções Python usando 
`types.FunctionDeclaration.from_callable(client=client, callable=your_function)`.

## Chamada de função com modelos de pensamento

Os modelos das séries Gemini 3 e 2.5 usam um processo interno de 
["raciocínio"](https://ai.google.dev/gemini-api/docs/thinking?hl=pt-br) para analisar as 
solicitações. Isso melhora significativamente o desempenho da chamada de função, permitindo que 
o modelo determine melhor quando chamar uma função e quais parâmetros usar. Como a API Gemini 
não tem estado, os modelos usam [assinaturas de 
pensamento](https://ai.google.dev/gemini-api/docs/thought-signatures?hl=pt-br) para manter o 
contexto em conversas de vários turnos.

Esta seção aborda o gerenciamento avançado de assinaturas de pensamento e só é necessária se 
você estiver criando solicitações de API manualmente (por exemplo, via REST) ou manipulando o 
histórico de conversas.

**Se você estiver usando os [SDKs da GenAI do 
Google](https://ai.google.dev/gemini-api/docs/libraries?hl=pt-br) (nossas bibliotecas oficiais), 
não será necessário gerenciar esse processo**. Os SDKs processam automaticamente as etapas 
necessárias, conforme mostrado no 
[exemplo](https://ai.google.dev/gemini-api/docs/function-calling?hl=pt-br#step-4) anterior.

### Gerenciar o histórico de conversas manualmente

Se você modificar o histórico de conversas manualmente, em vez de enviar a [resposta anterior 
completa](https://ai.google.dev/gemini-api/docs/function-calling?hl=pt-br#step-4), processe 
corretamente o `thought_signature` incluído na vez do modelo.

Siga estas regras para garantir que o contexto do modelo seja preservado:

- Sempre envie o `thought_signature` de volta ao modelo dentro do 
[`Part`](https://ai.google.dev/api?hl=pt-br#request-body-structure) original.
- **Sempre inclua o `id` exato do `function_call` no seu `function_response` para que a API possa 
mapear o resultado para a solicitação correta.**
- Não mescle um `Part` com uma assinatura e outro sem. Isso quebra o contexto posicional do 
pensamento.
- Não combine dois `Parts` que contenham assinaturas, porque as strings de assinatura não podem 
ser mescladas.

#### Assinaturas de raciocínio do Gemini 3

No Gemini 3, qualquer [`Part`](https://ai.google.dev/api?hl=pt-br#request-body-structure) de uma 
resposta do modelo pode conter uma assinatura de pensamento. Embora geralmente recomendemos 
retornar assinaturas de todos os tipos `Part`, transmitir assinaturas de pensamento é obrigatório 
para a chamada de função. A menos que você manipule o histórico de conversas manualmente, o SDK 
do Google GenAI vai processar as assinaturas de pensamento automaticamente.

Se você estiver manipulando o histórico de conversas manualmente, consulte a página [Assinaturas 
de pensamento](https://ai.google.dev/gemini-api/docs/thought-signatures?hl=pt-br) para 
orientações e detalhes completos sobre como lidar com assinaturas de pensamento no Gemini 3.

##### Como inspecionar assinaturas de raciocínio

Embora não seja necessário para a implementação, você pode inspecionar a resposta para ver o 
`thought_signature` para fins de depuração ou educacionais.

### Python

```
import base64
# After receiving a response from a model with thinking enabled
# response = client.models.generate_content(...)

# The signature is attached to the response part containing the function call
part = response.candidates[0].content.parts[0]
if part.thought_signature:
  print(base64.b64encode(part.thought_signature).decode("utf-8"))
```

### JavaScript

```
// After receiving a response from a model with thinking enabled
// const response = await ai.models.generateContent(...)

// The signature is attached to the response part containing the function call
const part = response.candidates[0].content.parts[0];
if (part.thoughtSignature) {
  console.log(part.thoughtSignature);
}
```

Saiba mais sobre as limitações e o uso de assinaturas de pensamento e sobre modelos de pensamento 
em geral na página 
[Pensamento](https://ai.google.dev/gemini-api/docs/thinking?hl=pt-br#signatures).

## Chamada de função paralela

Além da chamada de função única, você também pode chamar várias funções de uma só vez. 
Com a chamada de função paralela, é possível executar várias funções ao mesmo tempo, e ela 
é usada quando as funções não dependem umas das outras. Isso é útil em cenários como coleta 
de dados de várias fontes independentes, como recuperação de detalhes de clientes de diferentes 
bancos de dados ou verificação de níveis de inventário em vários armazéns ou execução de 
várias ações, como transformar seu apartamento em uma discoteca.

Quando o modelo inicia várias chamadas de função em um único turno, não é necessário 
retornar os objetos `function_result` na mesma ordem em que os objetos `function_call` foram 
recebidos. A API Gemini mapeia cada resultado de volta para a chamada correspondente usando o `id` 
da saída do modelo. Isso permite executar as funções de forma assíncrona e anexar os resultados 
à lista à medida que são concluídos.

### Python

```
power_disco_ball = {
    "name": "power_disco_ball",
    "description": "Powers the spinning disco ball.",
    "parameters": {
        "type": "object",
        "properties": {
            "power": {
                "type": "boolean",
                "description": "Whether to turn the disco ball on or off.",
            }
        },
        "required": ["power"],
    },
}

start_music = {
    "name": "start_music",
    "description": "Play some music matching the specified parameters.",
    "parameters": {
        "type": "object",
        "properties": {
            "energetic": {
                "type": "boolean",
                "description": "Whether the music is energetic or not.",
            },
            "loud": {
                "type": "boolean",
                "description": "Whether the music is loud or not.",
            },
        },
        "required": ["energetic", "loud"],
    },
}

dim_lights = {
    "name": "dim_lights",
    "description": "Dim the lights.",
    "parameters": {
        "type": "object",
        "properties": {
            "brightness": {
                "type": "number",
                "description": "The brightness of the lights, 0.0 is off, 1.0 is full.",
            }
        },
        "required": ["brightness"],
    },
}
```

### JavaScript

```
import { Type } from '@google/genai';

const powerDiscoBall = {
  name: 'power_disco_ball',
  description: 'Powers the spinning disco ball.',
  parameters: {
    type: Type.OBJECT,
    properties: {
      power: {
        type: Type.BOOLEAN,
        description: 'Whether to turn the disco ball on or off.'
      }
    },
    required: ['power']
  }
};

const startMusic = {
  name: 'start_music',
  description: 'Play some music matching the specified parameters.',
  parameters: {
    type: Type.OBJECT,
    properties: {
      energetic: {
        type: Type.BOOLEAN,
        description: 'Whether the music is energetic or not.'
      },
      loud: {
        type: Type.BOOLEAN,
        description: 'Whether the music is loud or not.'
      }
    },
    required: ['energetic', 'loud']
  }
};

const dimLights = {
  name: 'dim_lights',
  description: 'Dim the lights.',
  parameters: {
    type: Type.OBJECT,
    properties: {
      brightness: {
        type: Type.NUMBER,
        description: 'The brightness of the lights, 0.0 is off, 1.0 is full.'
      }
    },
    required: ['brightness']
  }
};
```

Configure o modo de chamada de função para permitir o uso de todas as ferramentas especificadas. 
Para saber mais, leia sobre [como configurar a chamada de 
função](https://ai.google.dev/gemini-api/docs/function-calling?hl=pt-br#function_calling_modes).

### Python

```
from google import genai
from google.genai import types

# Configure the client and tools
client = genai.Client()
house_tools = [
    types.Tool(function_declarations=[power_disco_ball, start_music, dim_lights])
]
config = types.GenerateContentConfig(
    tools=house_tools,
    automatic_function_calling=types.AutomaticFunctionCallingConfig(
        disable=True
    ),
    # Force the model to call 'any' function, instead of chatting.
    tool_config=types.ToolConfig(
        function_calling_config=types.FunctionCallingConfig(mode='ANY')
    ),
)

chat = client.chats.create(model="gemini-3.5-flash", config=config)
response = chat.send_message("Turn this place into a party!")

# Print out each of the function calls requested from this single call
print("Example 1: Forced function calling")
for fn in response.function_calls:
    args = ", ".join(f"{key}={val}" for key, val in fn.args.items())
    print(f"{fn.name}({args}) - ID: {fn.id}")
```

### JavaScript

```
import { GoogleGenAI } from '@google/genai';

// Set up function declarations
const houseFns = [powerDiscoBall, startMusic, dimLights];

const config = {
    tools: [{
        functionDeclarations: houseFns
    }],
    // Force the model to call 'any' function, instead of chatting.
    toolConfig: {
        functionCallingConfig: {
            mode: 'any'
        }
    }
};

// Configure the client
const ai = new GoogleGenAI({});

// Create a chat session
const chat = ai.chats.create({
    model: 'gemini-3.5-flash',
    config: config
});
const response = await chat.sendMessage({message: 'Turn this place into a party!'});

// Print out each of the function calls requested from this single call
console.log("Example 1: Forced function calling");
for (const fn of response.functionCalls) {
    const args = Object.entries(fn.args)
        .map(([key, val]) => \`${key}=${val}\`)
        .join(', ');
    console.log(\`${fn.name}(${args}) - ID: ${fn.id}\`);
}
```

Cada um dos resultados impressos reflete uma única chamada de função que o modelo solicitou. 
Para enviar os resultados, inclua as respostas na mesma ordem em que foram solicitadas.

O SDK do Python oferece suporte à [chamada automática de 
função](https://ai.google.dev/gemini-api/docs/function-calling?hl=pt-br#automatic_function_calling
_python_only), que converte automaticamente funções Python em declarações e processa o ciclo de 
execução e resposta da chamada de função para você. Confira um exemplo para o caso de uso de 
disco.

### Python

```
from google import genai
from google.genai import types

# Actual function implementations
def power_disco_ball_impl(power: bool) -> dict:
    """Powers the spinning disco ball.

    Args:
        power: Whether to turn the disco ball on or off.

    Returns:
        A status dictionary indicating the current state.
    """
    return {"status": f"Disco ball powered {'on' if power else 'off'}"}

def start_music_impl(energetic: bool, loud: bool) -> dict:
    """Play some music matching the specified parameters.

    Args:
        energetic: Whether the music is energetic or not.
        loud: Whether the music is loud or not.

    Returns:
        A dictionary containing the music settings.
    """
    music_type = "energetic" if energetic else "chill"
    volume = "loud" if loud else "quiet"
    return {"music_type": music_type, "volume": volume}

def dim_lights_impl(brightness: float) -> dict:
    """Dim the lights.

    Args:
        brightness: The brightness of the lights, 0.0 is off, 1.0 is full.

    Returns:
        A dictionary containing the new brightness setting.
    """
    return {"brightness": brightness}

# Configure the client
client = genai.Client()
config = types.GenerateContentConfig(
    tools=[power_disco_ball_impl, start_music_impl, dim_lights_impl]
)

# Make the request
response = client.models.generate_content(
    model="gemini-3.5-flash",
    contents="Do everything you need to this place into party!",
    config=config,
)

print("\nExample 2: Automatic function calling")
print(response.text)
# I've turned on the disco ball, started playing loud and energetic music, and dimmed the lights to 
50% brightness. Let's get this party started!
```

## Chamada de função composicional

A chamada de função composicional ou sequencial permite que o Gemini encadeie várias chamadas de 
função para atender a uma solicitação complexa. Por exemplo, para responder a "Qual é a 
temperatura no meu local atual?", a API Gemini pode primeiro invocar uma função 
`get_current_location()` seguida por uma função `get_weather()` que usa o local como parâmetro.

O exemplo a seguir demonstra como implementar a chamada de função composicional usando o SDK do 
Python e a chamada de função automática.

### Python

Este exemplo usa o recurso de chamada de função automática do SDK do Python `google-genai`. O 
SDK converte automaticamente as funções Python no esquema necessário, executa as chamadas de 
função quando solicitado pelo modelo e envia os resultados de volta para o modelo para concluir a 
tarefa.

```
import os
from google import genai
from google.genai import types

# Example Functions
def get_weather_forecast(location: str) -> dict:
    """Gets the current weather temperature for a given location."""
    print(f"Tool Call: get_weather_forecast(location={location})")
    # TODO: Make API call
    print("Tool Response: {'temperature': 25, 'unit': 'celsius'}")
    return {"temperature": 25, "unit": "celsius"}  # Dummy response

def set_thermostat_temperature(temperature: int) -> dict:
    """Sets the thermostat to a desired temperature."""
    print(f"Tool Call: set_thermostat_temperature(temperature={temperature})")
    # TODO: Interact with a thermostat API
    print("Tool Response: {'status': 'success'}")
    return {"status": "success"}

# Configure the client and model
client = genai.Client()
config = types.GenerateContentConfig(
    tools=[get_weather_forecast, set_thermostat_temperature]
)

# Make the request
response = client.models.generate_content(
    model="gemini-3.5-flash",
    contents="If it's warmer than 20°C in London, set the thermostat to 20°C, otherwise set it to 
18°C.",
    config=config,
)

# Print the final, user-facing response
print(response.text)
```

**Resposta esperada**

Ao executar o código, você vai ver o SDK orquestrando as chamadas de função. Primeiro, o modelo 
chama `get_weather_forecast`, recebe a temperatura e chama `set_thermostat_temperature` com o valor 
correto com base na lógica do comando.

```
Tool Call: get_weather_forecast(location=London)
Tool Response: {'temperature': 25, 'unit': 'celsius'}
Tool Call: set_thermostat_temperature(temperature=20)
Tool Response: {'status': 'success'}
OK. I've set the thermostat to 20°C.
```

### JavaScript

Este exemplo mostra como usar o SDK JavaScript/TypeScript para fazer chamadas de função 
composicionais usando um loop de execução manual.

```
import { GoogleGenAI, Type } from "@google/genai";

// Configure the client
const ai = new GoogleGenAI({});

// Example Functions
function get_weather_forecast({ location }) {
  console.log(\`Tool Call: get_weather_forecast(location=${location})\`);
  // TODO: Make API call
  console.log("Tool Response: {'temperature': 25, 'unit': 'celsius'}");
  return { temperature: 25, unit: "celsius" };
}

function set_thermostat_temperature({ temperature }) {
  console.log(
    \`Tool Call: set_thermostat_temperature(temperature=${temperature})\`,
  );
  // TODO: Make API call
  console.log("Tool Response: {'status': 'success'}");
  return { status: "success" };
}

const toolFunctions = {
  get_weather_forecast,
  set_thermostat_temperature,
};

const tools = [
  {
    functionDeclarations: [
      {
        name: "get_weather_forecast",
        description:
          "Gets the current weather temperature for a given location.",
        parameters: {
          type: Type.OBJECT,
          properties: {
            location: {
              type: Type.STRING,
            },
          },
          required: ["location"],
        },
      },
      {
        name: "set_thermostat_temperature",
        description: "Sets the thermostat to a desired temperature.",
        parameters: {
          type: Type.OBJECT,
          properties: {
            temperature: {
              type: Type.NUMBER,
            },
          },
          required: ["temperature"],
        },
      },
    ],
  },
];

// Prompt for the model
let contents = [
  {
    role: "user",
    parts: [
      {
        text: "If it's warmer than 20°C in London, set the thermostat to 20°C, otherwise set it 
to 18°C.",
      },
    ],
  },
];

// Loop until the model has no more function calls to make
while (true) {
  const result = await ai.models.generateContent({
    model: "gemini-3.5-flash",
    contents,
    config: { tools },
  });

  if (result.functionCalls && result.functionCalls.length > 0) {
    const functionCall = result.functionCalls[0];

    const { name, args } = functionCall;

    if (!toolFunctions[name]) {
      throw new Error(\`Unknown function call: ${name}\`);
    }

    // Call the function and get the response.
    const toolResponse = toolFunctions[name](args);

    const functionResponsePart = {
      name: functionCall.name,
      response: {
        result: toolResponse,
      },
      id: functionCall.id,
    };

    // Send the function response back to the model.
    contents.push({
      role: "model",
      parts: [
        {
          functionCall: functionCall,
        },
      ],
    });
    contents.push({
      role: "user",
      parts: [
        {
          functionResponse: functionResponsePart,
        },
      ],
    });
  } else {
    // No more function calls, break the loop.
    console.log(result.text);
    break;
  }
}
```

**Resposta esperada**

Ao executar o código, você vai ver o SDK orquestrando as chamadas de função. Primeiro, o modelo 
chama `get_weather_forecast`, recebe a temperatura e chama `set_thermostat_temperature` com o valor 
correto com base na lógica do comando.

```
Tool Call: get_weather_forecast(location=London)
Tool Response: {'temperature': 25, 'unit': 'celsius'}
Tool Call: set_thermostat_temperature(temperature=20)
Tool Response: {'status': 'success'}
OK. It's 25°C in London, so I've set the thermostat to 20°C.
```

A chamada de função composicional é um recurso nativo da [API 
Live](https://ai.google.dev/gemini-api/docs/live?hl=pt-br). Isso significa que a API Live pode 
processar a chamada de função de maneira semelhante ao SDK do Python.

### Python

```
# Light control schemas
turn_on_the_lights_schema = {'name': 'turn_on_the_lights'}
turn_off_the_lights_schema = {'name': 'turn_off_the_lights'}

prompt = """
  Hey, can you write run some python code to turn on the lights, wait 10s and then turn off the 
lights?
  """

tools = [
    {'code_execution': {}},
    {'function_declarations': [turn_on_the_lights_schema, turn_off_the_lights_schema]}
]

await run(prompt, tools=tools, modality="AUDIO")
```

### JavaScript

```
// Light control schemas
const turnOnTheLightsSchema = { name: 'turn_on_the_lights' };
const turnOffTheLightsSchema = { name: 'turn_off_the_lights' };

const prompt = \`
  Hey, can you write run some python code to turn on the lights, wait 10s and then turn off the 
lights?
\`;

const tools = [
  { codeExecution: {} },
  { functionDeclarations: [turnOnTheLightsSchema, turnOffTheLightsSchema] }
];

await run(prompt, tools=tools, modality="AUDIO")
```

## Modos de chamada de função

Com a API Gemini, você controla como o modelo usa as ferramentas fornecidas (declarações de 
função). Especificamente, é possível definir o modo em `function_calling_config`.

- `VALIDATED`: modo padrão para combinação de ferramentas (quando as ferramentas integradas ou 
saídas estruturadas também estão ativadas). O modelo é restrito a prever chamadas de função 
ou linguagem natural e garante a adesão ao esquema de função. Se `allowed_function_names` não 
for fornecido, o modelo vai escolher entre todas as declarações de função disponíveis. Se 
`allowed_function_names` for fornecido, o modelo vai escolher entre o conjunto de funções 
permitidas. Esse modo reduz as chamadas de função malformadas (em comparação com o modo `AUTO`).
- `AUTO`: modo padrão quando apenas a ferramenta "function\_declarations" está ativada. O modelo 
decide se quer gerar uma resposta de linguagem natural ou sugerir uma chamada de função com base 
no comando e no contexto.
- `ANY`: o modelo é restrito a sempre prever uma chamada de função e garante a adesão ao 
esquema de função. Se `allowed_function_names` não for especificado, o modelo poderá escolher 
qualquer uma das declarações de função fornecidas. Se `allowed_function_names` for fornecido 
como uma lista, o modelo só poderá escolher entre as funções dessa lista. Use esse modo quando 
precisar de uma resposta de chamada de função para cada comando (se aplicável).
- `NONE`: o modelo é *proibido* de fazer chamadas de função. Isso é equivalente a enviar uma 
solicitação sem declarações de função. Use isso para desativar temporariamente as chamadas de 
função sem remover as definições de ferramentas.

### Python

```
from google.genai import types

# Configure function calling mode
tool_config = types.ToolConfig(
    function_calling_config=types.FunctionCallingConfig(
        mode="ANY", allowed_function_names=["get_current_temperature"]
    )
)

# Create the generation config
config = types.GenerateContentConfig(
    tools=[tools],  # not defined here.
    tool_config=tool_config,
)
```

### JavaScript

```
import { FunctionCallingConfigMode } from '@google/genai';

// Configure function calling mode
const toolConfig = {
  functionCallingConfig: {
    mode: FunctionCallingConfigMode.ANY,
    allowedFunctionNames: ['get_current_temperature']
  }
};

// Create the generation config
const config = {
  tools: tools, // not defined here.
  toolConfig: toolConfig,
};
```

## Chamada automática de função (somente em Python)

Ao usar o SDK para Python, é possível fornecer funções do Python diretamente como ferramentas. 
O SDK converte essas funções em declarações, gerencia a execução da chamada de função e 
processa o ciclo de resposta para você. Defina a função com dicas de tipo e uma docstring. Para 
ter os melhores resultados, recomendamos usar [docstrings no estilo do 
Google](https://google.github.io/styleguide/pyguide.html#383-functions-and-methods). Em seguida, o 
SDK vai automaticamente:

1. Detectar respostas de chamada de função do modelo.
2. Chame a função Python correspondente no seu código.
3. Envie a resposta da função de volta ao modelo.
4. Retorne a resposta de texto final do modelo.

No momento, o SDK não analisa as descrições de argumentos nos slots de descrição de 
propriedade da declaração de função gerada. Em vez disso, ele envia a docstring inteira como a 
descrição da função de nível superior.

### Python

```
from google import genai
from google.genai import types

# Define the function with type hints and docstring
def get_current_temperature(location: str) -> dict:
    """Gets the current temperature for a given location.

    Args:
        location: The city and state, e.g. San Francisco, CA

    Returns:
        A dictionary containing the temperature and unit.
    """
    # ... (implementation) ...
    return {"temperature": 25, "unit": "Celsius"}

# Configure the client
client = genai.Client()
config = types.GenerateContentConfig(
    tools=[get_current_temperature]
)  # Pass the function itself

# Make the request
response = client.models.generate_content(
    model="gemini-3.5-flash",
    contents="What's the temperature in Boston?",
    config=config,
)

print(response.text)  # The SDK handles the function call and returns the final text
```

Para desativar a chamada automática de função, use:

### Python

```
config = types.GenerateContentConfig(
    tools=[get_current_temperature],
    automatic_function_calling=types.AutomaticFunctionCallingConfig(disable=True)
)
```

### Declaração automática de esquema de função

A API pode descrever qualquer um dos seguintes tipos. Os tipos `Pydantic` são permitidos, desde 
que os campos definidos neles também sejam compostos de tipos permitidos. Os tipos de dicionário 
(como `dict[str: int]`) não são bem compatíveis aqui. Não os use.

### Python

```
AllowedType = (
  int | float | bool | str | list['AllowedType'] | pydantic.BaseModel)
```

Para ver como é o esquema inferido, converta-o usando 
[`from_callable`](https://googleapis.github.io/python-genai/genai.html#genai.types.FunctionDeclarati
on.from_callable):

### Python

```
from google import genai
from google.genai import types

def multiply(a: float, b: float):
    """Returns a * b."""
    return a * b

client = genai.Client()
fn_decl = types.FunctionDeclaration.from_callable(callable=multiply, client=client)

# to_json_dict() provides a clean JSON representation.
print(fn_decl.to_json_dict())
```

## Uso de várias ferramentas: combine ferramentas integradas com chamadas de função

É possível ativar várias ferramentas, combinando as integradas com a chamada de função na 
mesma solicitação.

Os modelos do Gemini 3 podem combinar ferramentas integradas com a chamada de função pronta para 
uso, graças ao recurso de circulação de contexto da ferramenta. Leia a página sobre [Como 
combinar ferramentas integradas e chamadas de 
função](https://ai.google.dev/gemini-api/docs/tool-combination?hl=pt-br) para saber mais.

### Python

```
from google import genai
from google.genai import types

client = genai.Client()

getWeather = {
    "name": "getWeather",
    "description": "Gets the weather for a requested city.",
    "parameters": {
        "type": "object",
        "properties": {
            "city": {
                "type": "string",
                "description": "The city and state, e.g. Utqiaġvik, Alaska",
            },
        },
        "required": ["city"],
    },
}

response = client.models.generate_content(
    model="gemini-3.5-flash",
    contents="What is the northernmost city in the United States? What's the weather like there 
today?",
    config=types.GenerateContentConfig(
      tools=[
        types.Tool(
          google_search=types.ToolGoogleSearch(),  # Built-in tool
          function_declarations=[getWeather]       # Custom tool
        ),
      ],
      include_server_side_tool_invocations=True
    ),
)

history = [
    types.Content(
        role="user",
        parts=[types.Part(text="What is the northernmost city in the United States? What's the 
weather like there today?")]
    ),
    response.candidates[0].content,
    types.Content(
        role="user",
        parts=[types.Part(
            function_response=types.FunctionResponse(
                name="getWeather",
                response={"response": "Very cold. 22 degrees Fahrenheit."},
                id=response.candidates[0].content.parts[2].function_call.id
            )
        )]
    )
]

response_2 = client.models.generate_content(
    model="gemini-3.5-flash",
    contents=history,
    config=types.GenerateContentConfig(
      tools=[
        types.Tool(
          google_search=types.ToolGoogleSearch(),
          function_declarations=[getWeather]
        ),
      ],
      include_server_side_tool_invocations=True
    ),
)
```

### JavaScript

```
import { GoogleGenAI, Type } from '@google/genai';

const client = new GoogleGenAI({});

const getWeather = {
    name: "getWeather",
    description: "Get the weather in a given location",
    parameters: {
        type: "OBJECT",
        properties: {
            location: {
                type: "STRING",
                description: "The city and state, e.g. San Francisco, CA"
            }
        },
        required: ["location"]
    }
};

async function run() {
    const model = client.models.generateContent({
        model: "gemini-3.5-flash",
    });

    const tools = [
      { googleSearch: {} },
      { functionDeclarations: [getWeather] }
    ];
    const toolConfig = { includeServerSideToolInvocations: true };

    const result1 = await model.generateContent({
        contents: [{role: "user", parts: [{text: "What is the northernmost city in the United 
States? What's the weather like there today?"}]}],
        tools: tools,
        toolConfig: toolConfig,
    });

    const response1 = result1.response;
    const functionCallId = response1.candidates[0].content.parts.find(p => 
p.functionCall)?.functionCall?.id;

    const history = [
        {
            role: "user",
            parts:[{text: "What is the northernmost city in the United States? What's the weather 
like there today?"}]
        },
        response1.candidates[0].content,
        {
            role: "user",
            parts: [{
                functionResponse: {
                    name: "getWeather",
                    response: {response: "Very cold. 22 degrees Fahrenheit."},
                    id: functionCallId
                }
            }]
        }
    ];

    const result2 = await model.generateContent({
        contents: history,
        tools: tools,
        toolConfig: toolConfig,
    });
}

run();
```

Para modelos anteriores à série Gemini 3, use a [API 
Live](https://ai.google.dev/gemini-api/docs/live-api/tools?hl=pt-br).

## Respostas de funções multimodais

Para modelos da série Gemini 3, você pode incluir conteúdo multimodal nas partes de resposta da 
função que envia ao modelo. O modelo pode processar esse conteúdo multimodal na próxima vez 
para produzir uma resposta mais completa. Os seguintes tipos MIME são compatíveis com conteúdo 
multimodal em respostas de função:

- **Imagens**: `image/png`, `image/jpeg`, `image/webp`
- **Documentos**: `application/pdf`, `text/plain`

Para incluir dados multimodais em uma resposta de função, adicione-os como uma ou mais partes 
aninhadas na parte `functionResponse`. Cada parte multimodal precisa conter `inlineData`. Se você 
fizer referência a uma parte multimodal no campo estruturado `response`, ela precisará conter um 
`displayName` exclusivo.

Também é possível referenciar uma parte multimodal do campo `response` estruturado da parte 
`functionResponse` usando o formato de referência JSON `{"$ref": "<displayName>"}`. O modelo 
substitui a referência pelo conteúdo multimodal ao processar a resposta. Cada `displayName` só 
pode ser referenciado uma vez no campo `response` estruturado.

O exemplo a seguir mostra uma mensagem que contém um `functionResponse` para uma função chamada 
`get_image` e uma parte aninhada com dados de imagem com `displayName: "instrument.jpg"`. O campo 
`response` do `functionResponse` faz referência a esta parte da imagem:

### Python

```
from google import genai
from google.genai import types

import requests

client = genai.Client()

# This is a manual, two turn multimodal function calling workflow:

# 1. Define the function tool
get_image_declaration = types.FunctionDeclaration(
  name="get_image",
  description="Retrieves the image file reference for a specific order item.",
  parameters={
      "type": "object",
      "properties": {
          "item_name": {
              "type": "string",
              "description": "The name or description of the item ordered (e.g., 'instrument')."
          }
      },
      "required": ["item_name"],
  },
)
tool_config = types.Tool(function_declarations=[get_image_declaration])

# 2. Send a message that triggers the tool
prompt = "Show me the instrument I ordered last month."
response_1 = client.models.generate_content(
  model="gemini-3.5-flash",
  contents=[prompt],
  config=types.GenerateContentConfig(
      tools=[tool_config],
  )
)

# 3. Handle the function call
function_call = response_1.function_calls[0]
requested_item = function_call.args["item_name"]
print(f"Model wants to call: {function_call.name}")

# Execute your tool (e.g., call an API)
# (This is a mock response for the example)
print(f"Calling external tool for: {requested_item}")

function_response_data = {
  "image_ref": {"$ref": "instrument.jpg"},
}
image_path = "https://goo.gle/instrument-img"
image_bytes = requests.get(image_path).content
function_response_multimodal_data = types.FunctionResponsePart(
  inline_data=types.FunctionResponseBlob(
    mime_type="image/jpeg",
    display_name="instrument.jpg",
    data=image_bytes,
  )
)

# 4. Send the tool's result back
# Append this turn's messages to history for a final response.
history = [
  types.Content(role="user", parts=[types.Part(text=prompt)]),
  response_1.candidates[0].content,
  types.Content(
    role="user",
    parts=[
        types.Part.from_function_response(
          id=function_call.id,
          name=function_call.name,
          response=function_response_data,
          parts=[function_response_multimodal_data]
        )
    ],
  )
]

response_2 = client.models.generate_content(
  model="gemini-3.5-flash",
  contents=history,
  config=types.GenerateContentConfig(
      tools=[tool_config],
      thinking_config=types.ThinkingConfig(include_thoughts=True)
  ),
)

print(f"\nFinal model response: {response_2.text}")
```

### JavaScript

```
import { GoogleGenAI, Type } from '@google/genai';

const client = new GoogleGenAI({ apiKey: process.env.GEMINI_API_KEY });

// This is a manual, two turn multimodal function calling workflow:
// 1. Define the function tool
const getImageDeclaration = {
  name: 'get_image',
  description: 'Retrieves the image file reference for a specific order item.',
  parameters: {
    type: Type.OBJECT,
    properties: {
      item_name: {
        type: Type.STRING,
        description: "The name or description of the item ordered (e.g., 'instrument').",
      },
    },
    required: ['item_name'],
  },
};

const toolConfig = {
  functionDeclarations: [getImageDeclaration],
};

// 2. Send a message that triggers the tool
const prompt = 'Show me the instrument I ordered last month.';
const response1 = await client.models.generateContent({
  model: 'gemini-3.5-flash',
  contents: prompt,
  config: {
    tools: [toolConfig],
  },
});

// 3. Handle the function call
const functionCall = response1.functionCalls[0];
const requestedItem = functionCall.args.item_name;
console.log(\`Model wants to call: ${functionCall.name}\`);

// Execute your tool (e.g., call an API)
// (This is a mock response for the example)
console.log(\`Calling external tool for: ${requestedItem}\`);

const functionResponseData = {
  image_ref: { $ref: 'instrument.jpg' },
};

const imageUrl = "https://goo.gle/instrument-img";
const response = await fetch(imageUrl);
const imageArrayBuffer = await response.arrayBuffer();
const base64ImageData = Buffer.from(imageArrayBuffer).toString('base64');

const functionResponseMultimodalData = {
  inlineData: {
    mimeType: 'image/jpeg',
    displayName: 'instrument.jpg',
    data: base64ImageData,
  },
};

// 4. Send the tool's result back
// Append this turn's messages to history for a final response.
const history = [
  { role: 'user', parts: [{ text: prompt }] },
  response1.candidates[0].content,
  {
    role: 'user',
    parts: [
      {
        functionResponse: {
          id: functionCall.id,
          name: functionCall.name,
          response: functionResponseData,
          parts: [functionResponseMultimodalData]
        },
      },
    ],
  },
];

const response2 = await client.models.generateContent({
  model: 'gemini-3.5-flash',
  contents: history,
  config: {
    tools: [toolConfig],
    thinkingConfig: { includeThoughts: true },
  },
});

console.log(\`\nFinal model response: ${response2.text}\`);
```

### REST

```
IMG_URL="https://goo.gle/instrument-img"

MIME_TYPE=$(curl -sIL "$IMG_URL" | grep -i '^content-type:' | awk -F ': ' '{print $2}' | sed 
's/\r$//' | head -n 1)
if [[ -z "$MIME_TYPE" || ! "$MIME_TYPE" == image/* ]]; then
  MIME_TYPE="image/jpeg"
fi

# Check for macOS
if [[ "$(uname)" == "Darwin" ]]; then
  IMAGE_B64=$(curl -sL "$IMG_URL" | base64 -b 0)
elif [[ "$(base64 --version 2>&1)" = *"FreeBSD"* ]]; then
  IMAGE_B64=$(curl -sL "$IMG_URL" | base64)
else
  IMAGE_B64=$(curl -sL "$IMG_URL" | base64 -w0)
fi

curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash:generateContent" \
  -H "x-goog-api-key: $GEMINI_API_KEY" \
  -H 'Content-Type: application/json' \
  -X POST \
  -d '{
    "contents": [
      ...,
      {
        "role": "user",
        "parts": [
        {
            "functionResponse": {
              "name": "get_image",
              "id": "UNIQUE_CALL_ID_HERE",
              "response": {
                "image_ref": {
                  "$ref": "instrument.jpg"
                }
              },
              "parts": [
                {
                  "inlineData": {
                    "displayName": "instrument.jpg",
                    "mimeType":"'"$MIME_TYPE"'",
                    "data": "'"$IMAGE_B64"'"
                  }
                }
              ]
            }
          }
        ]
      }
    ]
  }'
```

## Chamada de função com saída estruturada

Para modelos da série Gemini 3, é possível usar a chamada de função com [saída 
estruturada](https://ai.google.dev/gemini-api/docs/structured-output?hl=pt-br). Isso permite que o 
modelo preveja chamadas de função ou saídas que aderem a um esquema específico. Como resultado, 
você recebe respostas com formatação consistente quando o modelo não gera chamadas de função.

## Protocolo de contexto de modelo (MCP)

O [Protocolo de Contexto de Modelo (MCP)](https://modelcontextprotocol.io/introduction) é um 
padrão aberto para conectar aplicativos de IA a ferramentas e dados externos. O MCP oferece um 
protocolo comum para que os modelos acessem o contexto, como funções (ferramentas), fontes de 
dados (recursos) ou comandos predefinidos.

Os SDKs do Gemini têm suporte integrado para o MCP, reduzindo o código boilerplate e oferecendo 
[chamada de função 
automática](https://ai.google.dev/gemini-api/docs/function-calling?hl=pt-br#automatic_function_call
ing_python_only) para ferramentas do MCP. Quando o modelo gera uma chamada de ferramenta do MCP, o 
SDK do cliente em Python e JavaScript pode executar automaticamente a ferramenta do MCP e enviar a 
resposta de volta ao modelo em uma solicitação subsequente, continuando esse loop até que o 
modelo não faça mais chamadas de ferramenta.

Confira um exemplo de como usar um servidor MCP local com o Gemini e o SDK `mcp`.

### Python

Verifique se a versão mais recente do SDK do [`mcp`](https://modelcontextprotocol.io/introduction) 
está instalada na plataforma escolhida.

```
pip install mcp
```

```
import os
import asyncio
from datetime import datetime
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client
from google import genai

client = genai.Client()

# Create server parameters for stdio connection
server_params = StdioServerParameters(
    command="npx",  # Executable
    args=["-y", "@philschmid/weather-mcp"],  # MCP Server
    env=None,  # Optional environment variables
)

async def run():
    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            # Prompt to get the weather for the current day in London.
            prompt = f"What is the weather in London in {datetime.now().strftime('%Y-%m-%d')}?"

            # Initialize the connection between client and server
            await session.initialize()

            # Send request to the model with MCP function declarations
            response = await client.aio.models.generate_content(
                model="gemini-3.5-flash",
                contents=prompt,
                config=genai.types.GenerateContentConfig(
                    temperature=0,
                    tools=[session],  # uses the session, will automatically call the tool
                    # Uncomment if you **don't** want the SDK to automatically call the tool
                    # automatic_function_calling=genai.types.AutomaticFunctionCallingConfig(
                    #     disable=True
                    # ),
                ),
            )
            print(response.text)

# Start the asyncio event loop and run the main function
asyncio.run(run())
```

### JavaScript

Verifique se a versão mais recente do SDK `mcp` está instalada na plataforma de sua escolha.

```
npm install @modelcontextprotocol/sdk
```

```
import { GoogleGenAI, FunctionCallingConfigMode , mcpToTool} from '@google/genai';
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";

// Create server parameters for stdio connection
const serverParams = new StdioClientTransport({
  command: "npx", // Executable
  args: ["-y", "@philschmid/weather-mcp"] // MCP Server
});

const client = new Client(
  {
    name: "example-client",
    version: "1.0.0"
  }
);

// Configure the client
const ai = new GoogleGenAI({});

// Initialize the connection between client and server
await client.connect(serverParams);

// Send request to the model with MCP tools
const response = await ai.models.generateContent({
  model: "gemini-3.5-flash",
  contents: \`What is the weather in London in ${new Date().toLocaleDateString()}?\`,
  config: {
    tools: [mcpToTool(client)],  // uses the session, will automatically call the tool
    // Uncomment if you **don't** want the sdk to automatically call the tool
    // automaticFunctionCalling: {
    //   disable: true,
    // },
  },
});
console.log(response.text)

// Close the connection
await client.close();
```

### Limitações com suporte integrado ao MCP

O suporte integrado ao MCP é um recurso 
[experimental](https://ai.google.dev/gemini-api/docs/models?hl=pt-br#preview) nos nossos SDKs e tem 
as seguintes limitações:

- Somente ferramentas são aceitas, não recursos nem comandos
- Ele está disponível para os SDKs Python e JavaScript/TypeScript.
- Mudanças interruptivas podem ocorrer em versões futuras.

A integração manual de servidores MCP é sempre uma opção se esses limites afetarem o que você 
está criando.

## Modelos compatíveis

Esta seção lista os modelos e os recursos de chamada de função deles. Modelos experimentais 
não estão incluídos. Confira uma visão geral completa dos recursos na página [Visão geral do 
modelo](https://ai.google.dev/gemini-api/docs/models?hl=pt-br).

| Modelo | Chamadas de função | Chamada de função paralela | Chamada de função composicional |
| --- | --- | --- | --- |
| [Pré-lançamento do Gemini 3.1 
Pro](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-pro-preview?hl=pt-br) | ✔️ | 
✔️ | ✔️ |
| [Gemini 3.1 
Flash-Lite](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-lite?hl=pt-br) | ✔️ | 
✔️ | ✔️ |
| [Gemini 3.5 Flash](https://ai.google.dev/gemini-api/docs/models/gemini-3.5-flash?hl=pt-br) | 
✔️ | ✔️ | ✔️ |
| [Gemini 2.5 Pro](https://ai.google.dev/gemini-api/docs/models/gemini-2.5-pro?hl=pt-br) | ✔️ | 
✔️ | ✔️ |
| [Gemini 2.5 Flash](https://ai.google.dev/gemini-api/docs/models/gemini-2.5-flash?hl=pt-br) | 
✔️ | ✔️ | ✔️ |
| [Gemini 2.5 
Flash-Lite](https://ai.google.dev/gemini-api/docs/models/gemini-2.5-flash-lite?hl=pt-br) | ✔️ | 
✔️ | ✔️ |

## Práticas recomendadas

- **Descrições de funções e parâmetros**:seja extremamente claro e específico nas 
descrições. O modelo depende deles para escolher a função correta e fornecer argumentos 
adequados.
- **Nomenclatura**:use nomes de função descritivos (sem espaços, pontos ou traços).
- **Tipagem forte**:use tipos específicos (inteiro, string, enum) para parâmetros e reduza os 
erros. Se um parâmetro tiver um conjunto limitado de valores válidos, use uma enumeração.
- **Seleção de ferramentas**:embora o modelo possa usar um número arbitrário de ferramentas, 
fornecer muitas pode aumentar o risco de selecionar uma ferramenta incorreta ou inadequada. Para 
melhores resultados, forneça apenas as ferramentas relevantes para o contexto ou a tarefa, 
mantendo o conjunto ativo em um máximo de 10 a 20. Considere a seleção dinâmica de ferramentas 
com base no contexto da conversa se você tiver um grande número total de ferramentas.
- **Engenharia de comando**:
	- Forneça contexto: diga ao modelo qual é a função dele (por exemplo, "Você é um 
assistente de clima útil").
		- Dê instruções: especifique como e quando usar funções (por exemplo, "Não 
adivinhe datas. Sempre use uma data futura para previsões").
		- Incentive o esclarecimento: instrua o modelo a fazer perguntas de esclarecimento, 
se necessário.
		- Consulte [Fluxos de trabalho de 
agentes](https://ai.google.dev/gemini-api/docs/prompting-strategies?hl=pt-br#agentic-workflows) 
para mais estratégias de criação desses comandos. Confira um exemplo de uma [instrução do 
sistema](https://ai.google.dev/gemini-api/docs/prompting-strategies?hl=pt-br#agentic-si-template) 
testada.
- **Temperatura**:use uma temperatura baixa (por exemplo, 0) para chamadas de função mais 
deterministas e confiáveis.
- **Validação**:se uma chamada de função tiver consequências significativas (por exemplo, 
fazer um pedido), valide a chamada com o usuário antes de executá-la.
- **Verifique o motivo da conclusão**:sempre verifique o 
[`finishReason`](https://ai.google.dev/api/generate-content?hl=pt-br#FinishReason) na resposta do 
modelo para lidar com casos em que ele não gerou uma chamada de função válida.
- **Tratamento de erros**: implemente um tratamento de erros robusto nas suas funções para lidar 
com entradas inesperadas ou falhas de API. Retornar mensagens de erro informativas que o modelo 
pode usar para gerar respostas úteis ao usuário.
- **Segurança**:tenha cuidado ao chamar APIs externas. Use mecanismos de autenticação e 
autorização adequados. Evite expor dados sensíveis em chamadas de função.
- **Limites de tokens**:as descrições e os parâmetros de função são contabilizados no limite 
de tokens de entrada. Se você estiver atingindo os limites de token, considere limitar o número 
de funções ou o tamanho das descrições, divida tarefas complexas em conjuntos de funções 
menores e mais focados.
- **Combinação de bash e ferramentas personalizadas**: para quem cria com uma combinação de 
bash e ferramentas personalizadas, o pré-lançamento do Gemini 3.1 Pro vem com um endpoint 
separado disponível pela API chamado 
[`gemini-3.1-pro-preview-customtools`](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-pro-p
review?hl=pt-br#gemini-31-pro-preview-customtools).

## Observações e limitações:

- Posicionamento das partes de uma chamada de função: ao usar declarações de função 
personalizadas [com ferramentas 
integradas](https://ai.google.dev/gemini-api/docs/tool-combination?hl=pt-br) (como a Pesquisa 
Google), o modelo pode retornar uma combinação de partes `functionCall`, `toolCall` e 
`toolResponse` em uma única interação. Por isso, não suponha que o `functionCall` sempre será 
o último item na matriz de partes. Se você estiver analisando manualmente a resposta JSON, sempre 
itere pela matriz de partes em vez de confiar na posição.
- Apenas um [subconjunto do esquema 
OpenAPI](https://ai.google.dev/api/caching?hl=pt-br#FunctionDeclaration) é compatível.
- No modo `ANY`, a API pode rejeitar esquemas muito grandes ou profundamente aninhados. Se 
encontrar erros, tente simplificar os parâmetros da função e os esquemas de resposta encurtando 
os nomes das propriedades, reduzindo o aninhamento ou limitando o número de declarações de 
função.
- Os tipos de parâmetros compatíveis em Python são limitados.
- A chamada automática de função é um recurso exclusivo do SDK Python.
