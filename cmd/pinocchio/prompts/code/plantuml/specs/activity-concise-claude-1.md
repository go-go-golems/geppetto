Here is a simplified plantuml activity diagram specification focusing on the key features:

# Activity Diagram

Activities are denoted by `:` labels ending in `;`. They are linked in order of definition.

```
:Activity 1;
:Activity 2;
```

## Conditionals

Use `if`, `then`, `else` for conditionals:

```
if (condition) then
  :Activity 1;
else
  :Activity 2;
endif
```

## Loops

`repeat` and `repeat while` denote loops:

```
repeat
  :Activity 1;
repeat while (condition)
```

## Parallel Activities

`fork` starts parallel activities, `end fork` joins them:

```
fork
  :Activity 1;
fork
  :Activity 2;
end fork
```

## Partitions

Group activities using `partition`:

```
partition "Name" {
  :Activity 1;
  :Activity 2;
}
```

## Swimlanes

Denote swimlanes with `|`:

```
|Swimlane1|
:Activity 1;
|Swimlane2|
:Activity 2;
```

## Notes

Add notes with the `note` keyword:

```
note right: Text
```

So in summary, focus on:

- Basic activities
- Conditionals
- Loops
- Parallel activities
- Partitions
- Swimlanes
- Notes
