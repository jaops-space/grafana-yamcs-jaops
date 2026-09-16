# possible discussion issues

these are just possible github issues to maybe open later.

the idea is not to make a public roadmap or promise that we are building all of this for free. it is more to have discussion triggers, see if people are interested, and maybe let companies/users say what they actually need.

## feature: query multiple parameters per grafana query

right now, one grafana query is mostly one yamcs parameter.

it could be nice to support multiple parameters in the same query, so the parameter picker could allow selecting more than one parameter.

another related thing is aggregates. maybe instead of manually typing aggregate paths, we could let users navigate them with dropdowns.

things to discuss:

- should one query return multiple parameters?
- should this be multiple fields in one frame, or multiple frames?
- how should this work for live queries?
- how should this work for aggregates?
- would this make dashboards easier or just make queries more confusing?

this is mostly to see if people actually want this and what shape would make sense.

## feature: use actual ffmpeg/video streaming for image streaming

right now image streaming is done by fetching image urls.

that works, but it can be sensitive to browser cache / refresh behavior, and it does not feel like real video streaming.

one possible direction is to use actual ffmpeg/video streaming, kind of like the lunar rover dashboard. this could make image streaming nicer and more reliable.

this might need a new panel instead of changing the existing image panel.

things to discuss:

- should image streaming stay simple with image urls?
- should we add a real video streaming panel?
- can ffmpeg help here?
- how would this work with yamcs image products?
- is this useful enough, or too specific?

## feature: spacecraft-specific 3d / ground visualization panels

it could be interesting to have more mission-style visual panels.

for example:

- 3d spacecraft view
- spacecraft attitude view
- moon / planet ground view
- position or path view
- some kind of interactive telemetry-driven scene

the goal would be to make dashboards nicer and more useful for mission demos, but we should be careful not to build something too specific unless someone actually needs it.

this is mainly a discussion issue to see what people would expect from spacecraft-specific visualization.

## feature: advanced variable setting panel

the variable setting panel could maybe support more advanced values.

for example, it could parse expressions, do simple computations, or maybe use parameter values as inputs.

possible ideas:

- write simple expressions instead of only raw values
- use grafana variables in expressions
- maybe link inputs to current yamcs parameter values
- preview the computed value before applying it
- validate the final value before sending it

not sure if this belongs in the plugin or if it becomes too much, but it could be useful for more advanced dashboards.

## feature: canvas-like commanding panel

right now, if someone wants a command dashboard with many buttons, the layout is mostly done by putting command buttons in multiple panels.

it could be useful to have something more like grafana canvas, where you can place multiple command buttons around in one visual layout.

unfortunately, extending grafana canvas itself does not really seem like an option.

another possible path is to document how to use grafana canvas buttons that send HTTP requests, and explain exactly what endpoint/payload format is needed to trigger commands through the plugin backend or yamcs.

things to discuss:

- should we build a dedicated commanding layout panel?
- should we just document how to use grafana canvas buttons?
- how should confirmation / safety / verification be shown?
- how should command status feedback work?
- what is the cleanest way to place many commands on one dashboard?

## feature: parameter health overview panel

it could be useful to have one panel that gives a quick health overview of many parameters.

for example:

- latest value
- alarm state
- expired/stale state
- timestamp
- grouped by subsystem
- only show problems mode

this would be for dashboards where users do not want to inspect 20 time series just to know if something is wrong.

## feature: command presets / command templates

some commands are probably sent often with the same arguments.

it could be useful to have command presets or templates.

possible ideas:

- save common argument sets
- expose presets in the commanding panel
- allow grafana variables in command arguments
- require confirmation for risky presets
- link presets with command history

main question is whether this makes commanding easier without making it less safe.

## feature: better parameter search and discovery

when the yamcs MDB gets bigger, finding the right parameter can be annoying.

possible improvements:

- fuzzy search
- filter by subsystem / namespace / type / unit
- show descriptions
- show if a parameter is numeric, enum, string, boolean, aggregate, etc.
- recently used parameters
- pinned parameters
- better browsing of nested space systems

this is probably one of the more practical UX things, but still worth discussing first.

## feature: enum / state visualization helpers

yamcs has a lot of enums, booleans, modes, states, etc.

grafana can already display values, but maybe the plugin could make these easier to use.

possible ideas:

- status badges for enum values
- mode timeline
- boolean indicator
- color mapping based on yamcs metadata
- alarm-aware enum display

question is whether we need custom panels/helpers, or just better defaults/docs for existing grafana panels.

## feature: generate demo dashboards from yamcs metadata

it could be useful to generate starter dashboards from the yamcs MDB.

possible ideas:

- select a space system and generate a dashboard
- create panels based on parameter type / unit / alarms
- group parameters by subsystem
- include command panels for available commands
- generate a nice demo dashboard for yamcs-quickstart

this could help new users quickly try the plugin without already knowing what parameters to plot.
