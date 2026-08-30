# Quality checks

Quality checks help you decide what deserves a closer look. They do not replace code review.

## The basic workflow

1. Choose a quality profile.
2. Run the analysis.
3. Read the findings.
4. Fix real problems.
5. Review signals and decide whether they matter.
6. Baseline findings that are understood and accepted for now.
7. Run the analysis again.

The last step matters. A profile is only useful when its report is attached to the model you are reading.

## Two kinds of checks

### Exact checks

An exact check uses a defined measurement and comparison.

~~~text
File has 620 lines.
Profile limit is 500 lines.
620 is greater than 500.
Result: active finding.
~~~

This tells you exactly why the finding exists. It does not tell you whether the file is bad. A generated file, a parser, and a small application may have different reasons for being long.

### Advisory signals

An advisory signal notices a shape that may deserve review.

~~~text
BuildResult has 4 concrete dependencies.
The profile threshold is 3.
Result: review signal.
~~~

The signal is real: the structure was observed. The conclusion is not automatic: the tool cannot know whether the concrete dependencies are intentional.

## What the status words mean

| Status | Plain meaning | What to do |
| --- | --- | --- |
| Observed | The required facts were available. | Read the result. |
| Partial | Some subjects or facts were missing. | Treat the result as incomplete. |
| Unsupported | This analyzer does not provide the required fact. | Use an analyzer or feature that provides it, or leave it off. |
| Not evaluable | The rule exists, but this report cannot calculate it. | Check the reason and the required input. |
| Baseline | You reviewed this exact finding and accepted it for now. | Revisit it when the code or rule version changes. |

“Unsupported” does not mean “the rule is false.” It means the current input cannot supply the data.

## The number beside Observed

If the viewer says **13 of 25 checks observed**, it means:

- 25 rules are selected in the profile;
- 13 rules had enough data to evaluate;
- the remaining rules were unsupported or not evaluable for this report.

It is a coverage count. It is not a score.

## A useful question

Do not ask only “How many findings are there?” Ask:

> “Which findings are exact, which are review signals, and what evidence supports each one?”

Read [Findings and coverage](/quality/findings) for the answer.
