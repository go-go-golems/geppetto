PlantUML's activity diagrams provide a user-friendly and efficient way to visualize processes. The new syntax doesn't require Graphviz installation, simplifying the setup process. It's also easier to manage and maintain your diagrams. Here are some key features and their simplified examples:

1. **Simple Action**: Activities are labeled starting with `:` and ending with `;`. They are implicitly linked in their definition order.
    ```plantuml
    @startuml
    :Hello world;
    :This is defined on several **lines**;
    @enduml
    ```

2. **Start/Stop/End**: Use `start`, `stop`, and `end` keywords to denote the beginning and the end of a diagram.
    ```plantuml
    @startuml
    start
    :Hello world;
    :This is defined on several **lines**;
    stop
    @enduml
    ```

3. **Conditional**: Use `if`, `then`, and `else` keywords to put tests in your diagram.
    ```plantuml
    @startuml
    start
    if (Graphviz installed?) then (yes)
      :process all diagrams;
    else (no)
      :process only sequence and activity diagrams;
    endif
    stop
    @enduml
    ```

4. **Switch and Case**: Use `switch`, `case`, and `endswitch` keywords to put switch in your diagram.
    ```plantuml
    @startuml
    start
    switch (test?)
    case ( condition A )
      :Text 1;
    case ( condition B ) 
      :Text 2;
    endswitch
    stop
    @enduml
    ```

5. **Repeat Loop**: Use `repeat` and `repeat while` keywords to have repeat loops.
    ```plantuml
    @startuml
    start
    repeat
      :read data;
      :generate diagrams;
    repeat while (more data?) is (yes)
    stop
    @enduml
    ```

6. **While Loop**: Use `while` and `endwhile` keywords to have while loop.
    ```plantuml
    @startuml
    start
    while (data available?)
      :read data;
      :generate diagrams;
    endwhile
    stop
    @enduml
    ```

7. **Parallel Processing**: Use `fork`, `fork again`, and `end fork` keywords to denote parallel processing.
    ```plantuml
    @startuml
    start
    fork
      :action 1;
    fork again
      :action 2;
    end fork
    stop
    @enduml
    ```

8. **Split Processing**: Use `split`, `split again`, and `end split` keywords to denote split processing.
    ```plantuml
    @startuml
    start
    split
       :A;
    split again
       :B;
    end split
    stop
    @enduml
    ```

9. **Notes**: Use `note` keyword to add notes to your diagram.
    ```plantuml
    @startuml
    start
    :foo1;
    note right: This is a note
    :foo2;
    stop
    @enduml
    ```

10. **Colors**: Use `#color` to specify a color for some activities.
    ```plantuml
    @startuml
    start
    :starting progress;
    #HotPink:reading configuration files;
    #AAAAAA:ending of the process;
    @enduml
    ```

11. **Lines without Arrows**: Use `skinparam ArrowHeadColor none` to connect activities using lines only, without arrows.
    ```plantuml
    @startuml
    skinparam ArrowHeadColor none
    start
    :Hello world;
    :This is defined on several **lines**;
    stop
    @enduml
    ```

12. **Arrows**: Use `->` notation to add texts to arrow, and change their color.
    ```plantuml
    @startuml
    :foo1;
    -> You can put text on arrows;
    :foo2;
    @enduml
    ```

13. **Connector**: Use parentheses to denote connector.
    ```plantuml
    @startuml
    :Some activity;
    (A)
    detach
    (A)
    :Other activity;
    @enduml
    ```

14. **Grouping or Partition**: Use `group` or `partition` keywords to group activities together.
    ```plantuml
    @startuml
    start
    group Initialization 
        :read config file;
        :init internal variable;
    end group
    stop
    @enduml
    ```

15. **Swimlanes**: Use pipe `|` to define swimlanes.
    ```plantuml
    @startuml
    |Swimlane1|
    start
    :foo1;
    |Swimlane2|
    :foo2;
    stop
    @enduml
    ```

Here is a medium length diagram example:

```plantuml
@startuml
start
:ClickServlet.handleRequest();
:new page;
if (Page.onSecurityCheck) then (true)
  :Page.onInit();
  if (isForward?) then (no)
    :Process controls;
    if (continue processing?) then (no)
      stop
    endif
    if (isPost?) then (yes)
      :Page.onPost();
    else (no)
      :Page.onGet();
    endif
    :Page.onRender();
  endif
else (false)
endif
if (do redirect?) then (yes)
  :redirect process;
else
  if (do forward?) then (yes)
    :Forward request;
  else (no)
    :Render page template;
  endif
endif
stop
@enduml
``