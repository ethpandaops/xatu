## Create Plan template
This template describes the steps to create a plan for a given task. This DOES NOT include the actual issue description -- this will be provided by the user at a later stage. You will be given the issue description shortly. When instructed to create the plan, you must break down the task in to smaller tasks.

General guidelines:
- YOU MUST create a file in `./ai_plans/$issue_description.md` with the plan for the given task.
- DO NOT implement any code changes. Writing code in the plan is fine, but it must be in the form of a code block, and MUST NOT alter the codebase in any way.
- The plan MUST be in the form of a markdown file and be syntactically correct.
- The plan MUST be in the form of a plan, not a description of the task.
- The plan MUST provide as much detail as required to complete the task without any ambiguity. It should include relevent context, dependencies, and any other information that would be relevant to someone who is not the original author.

### Step 1: Gather information
- Gather relevant information. Look in the ./llms` folder for rules. Look for CLAUDE.md files throughout the codebase. 
- If you have it enabled, you can search the internet for additional information via MCP. For example, using Perplexity.

### Step 2: Ask the user for additional information
- This is an OPTIONAL step. If you do not need any additional information, you can skip this step. It is VERY important that you do not make any assumptions, so ask the user if you aren't certain. 

### Step 3: Build the plan
- Build out a detailed plan for the task. This MUST include the following:
  - An executive summary of the change, the problem it is solving, and the context of the task.
    - Include relevent files and code snippets to help with context.
    - Include a "BIG PICTURE" overview of the task.
    - IF relevent, include mermaid diagrams to help with context and the overall plan.
  - A list of assumptions that will be made to complete the task.
  - A detailed list of tasks that need to be completed to complete the task. 
    - The list should be granular and detailed, with a clear description of the task and the expected outcome.
    - MUST be in the form of a checklist.
    - MUST include a list of dependencies for each task.
    - MUST include any relevent commands that need to be run to complete the task.
    - MUST include any relevent links to documentation that will help complete the task.
    - MAY include the exact code that needs to be written to complete the task.

#### Format
````
# Plan for $task

## Executive Summary

## Detailed Plan

## Dependencies

## Assumptions

## Detailed Breakdown

## Tasks

## Context

## Assumptions

````

