---
Title: Least-to-Most prompting
Slug: least-to-most
Short: Use pinocchio and a few templates to test least-to-most prompting
Topics:
- prompt-engineering
- least-to-most
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Application
---

You can make use of pinocchio's templating capabilities to quickly run 
pretty large scale experiments. In this example, we will be examining
least-to-most prompting as described in 
"[Least-to-Most Prompting Enables Complex Reasoning in Large Language Models](https://arxiv.org/abs/2205.10625)"
by Zhou et al, 2022.

Least-to-Most Prompting builds upon the idea of Chain-of-Thought prompting 
and takes it a step further.
It is a technique inspired by real-world educational strategies for children.

As in CoT-Prompting,
the problem to be solved is decomposed in a set of subproblems that build upon each other.
In a second step, these subproblems are solved individually.
However, contrary to chain of thought, the solution of previous
subproblems is fed into the next question.

## Using pinocchio's example-driven template

To examine this problem using pinocchio, we use a fairly simple but extremely useful prompt 
template called `example-driven`, which can be reached
by using `pinocchio prompts example simple example-driven`.

```yaml
name: example-driven
short: A simple prompt showing how to use n-shot prompting
flags:
  - name: question
    type: string
    help: The question to ask
    required: true
  - name: problem
    type: string
    help: The problem to solve
    required: false
  - name: examples
    type: objectListFromFile
    required: true
prompt: |
  {{ if .problem -}}
  Problem: {{ .problem }}
  {{- end }}
  {{ range $i, $example := .examples }}
  Q: {{ $example.question }}
  A: {{ $example.answer }}
  {{ end }}
  
  Q: {{ .question }}
  A:%   
```

This template requires a list of questions in a yaml file (using the `objectListFromFile` glazed 
parameter type), as well as a question. Optionally, we can preface the series of example shots
with a problem statement.

## Solving letter concatenation 

We are now going to examine the test problem of concatenating the last letters of a sequence of
words.


### Standard few-shot prompting

We start by using the simplest way of few-shot prompting, by just giving the words and the answer.

```yaml
- question: think, machine
  answer: ke
- question: learning, reasoning, generalization
  answer: ggn
- question: artificial, intelligence
  answer: le
- question: transformer, language, vision
  answer: ren%  
```

We can call pinocchio and look at the resulting prompt:

```
❯ pinocchio prompts examples simple example-driven \
     --examples ./examples/letter-concatenation/standard-prompting.yaml \
     --question "barley, silk" \
     --print-prompt


Q: think, machine
A: ke

Q: learning, reasoning, generalization
A: ggn

Q: artificial, intelligence
A: le

Q: transformer, language, vision
A: ren


Q: barley, silk
A:

```

If we run this, we get pretty poor results already for just 2 words, 
even with bigger models such as text-davinci-003 or code-davinci-002. 
We pass in the "Q:" stop word to avoid having the model parrot more questions
as a response.

``` 
❯ pinocchio prompts examples simple example-driven \
    --examples ./examples/letter-concatenation/standard-prompting.yaml \
    --question "barley, silk" \
    --openai-stop "Q:"
 ks
 
❯ pinocchio prompts examples simple example-driven \
    --examples ./examples/letter-concatenation/standard-prompting.yaml \
    --question "barley, silk" \
    --openai-stop "Q:" \
    --openai-engine code-davinci-002
 zh
 ```

Not superbly useful.

### Chain-of-thought prompting

We can now try to use chain-of-thought prompting to solve this problem.

```yaml
- question: think, machine
  answer: >
    The last letter of "think" is "k".
    The last letter of "machine" is "e".
    So "think, machine" is "ke".
- question: learning, reasoning, generalization
  answer: >
    The last letter of "learning" is "g".
    The last letter of "reasoning" is "n".
    The last letter of "generalization" is "n".
    So "learning, reasoning, generalization" is "ggn".
- question: artificial, intelligence
  answer: >
    The last letter of "artificial" is "l".
    The last letter of "intelligence" is "e".
    So "artificial, intelligence" is "le".
- question: transformer, language, vision
  answer: >
    The last letter of "transformer" is "r".
    The last letter of "language" is "e".
    The last letter of "vision" is "n".
    So "transformer, language, vision" is "ren".
```

This leads to significantly better results, but still runs into problems when assembling
the final steps together.

```
❯ pinocchio prompts examples simple example-driven \
    --examples ./examples/letter-concatenation/chain-of-thought.yaml \
    --question "barley, silk, brands, slack"   \ 
    --openai-engine code-davinci-002 \ 
    --openai-stop "Q:"             
 The last letter of "barley" is "y".
 The last letter of "silk" is "k".
 The last letter of "brands" is "s".
 The last letter of "slack" is "k".
 So "barley, silk, brands, slack" is "yksk".
 
❯ pinocchio prompts examples simple example-driven \
      --examples ./examples/letter-concatenation/chain-of-thought.yaml \
      --question "barley, silk, brands, slack, programming, rocks, guitars, heavy, metal, boing, foobar, zilch" \
      --openai-engine code-davinci-002 \
      --openai-stop "Q:"
      
 The last letter of "barley" is "y".
 The last letter of "silk" is "k".
 The last letter of "brands" is "s".
 The last letter of "slack" is "k".
 The last letter of "programming" is "g".
 The last letter of "rocks" is "s".
 The last letter of "guitars" is "s".
 The last letter of "heavy" is "y".
 The last letter of "metal" is "l".
 The last letter of "boing" is "g".
 The last letter of "foobar" is "r".
 The last letter of "zilch" is "h".
 So "barley, silk, brands, slack, programming, rocks, guitars, heavy, metal, boing, foobar, zilch" 
 is "yksksgsyslgrh".
```

It is still wrong on longer words, but at least significantly better than already failing with 2 words.

### Least-to-Most prompting

The performance on longer words can significantly be improved by using least to most prompting,
where the examples show that the concatenation can be accumulated word by word.

```yaml
- question: think, machine
  answer: >
    The last letter of "think" is "k".
    The last letter of "machine" is "e".
    Concatenating "k" and "e" gives "ke".
    So "think, machine" output "ke".
- question: think, machine, learning
  answer: >
    "think, machine" outputs "ke".
    The last letter of "learning" is "g".
    Concatenating "ke" and "g" gives "keg".
    So "think, machine, keg" is "keg".
- question: transformer, language
  answer: >
    The last letter of "transformer" is "r".
    The last letter of "language" is "e".
    Concatenating "r" and "e" gives "re".
    So "transformer, language" is "re".
- question: transformer, language, vision
  answer: >
    "transformer, language" outputs "re".
    The last letter of "vision" is "n".
    Concatenating "re" and "n" gives "ren".
    So "transformer, language, vision" is "ren".
```

This leads to much better results, even on longer words sequences, with text-davinci-002 and text-davinci-003.

```
❯ pinocchio prompts examples simple example-driven \ 
        --examples ./examples/letter-concatenation/least-to-most.yaml \ 
        --question "barley, silk, brands, slack, programming, rocks, guitars, heavy, metal, boing, foobar, zilch"   \ 
        --openai-engine text-davinci-003 \ 
        --openai-max-response-tokens 1024
 The last letter of "barley" is "y".
  The last letter of "silk" is "k".
 Concatenating "y" and "k" gives "yk".
 The last letter of "brands" is "s".
 Concatenating "yk" and "s" gives "yks".
 The last letter of "slack" is "k".
 Concatenating "yks" and "k" gives "yksk".
 The last letter of "programming" is "g".
 Concatenating "yksk" and "g" gives "ykskg".
 The last letter of "rocks" is "s".
 Concatenating "ykskg" and "s" gives "ykskgs".
 The last letter of "guitars" is "s".
 Concatenating "ykskgs" and "s" gives "ykskgss".
 The last letter of "heavy" is "y".
 Concatenating "ykskgss" and "y" gives "ykskgssy".
 The last letter of "metal" is "l".
 Concatenating "ykskgssy" and "l" gives "ykskgssyl".
 The last letter of "boing" is "g".
 Concatenating "ykskgssyl" and "g" gives "ykskgssylg".
 The last letter of "foobar" is "r".
 Concatenating "ykskgssylg" and "r" gives "ykskgssylgr".
 The last letter of "zilch" is "h".
 Concatenating "ykskgssylgr" and "h" gives "ykskgssylgrh".
 So "barley, silk, brands, slack, programming, rocks, guitars, heavy, metal, boing, foobar, zilch" 
 is "ykskgssylgrh".
```