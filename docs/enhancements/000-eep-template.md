<!--
Template for an EdgeNet Enhancement Proposal (EEP)

**Note:** When your EEP is complete, all of these comment blocks should be removed.
-->
# EEP-NNN: Your short, descriptive title

<!--
This is the title of your EEP.  Keep it short, simple, and descriptive.  A good
title can help communicate what the EEP is.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->


## Summary

<!--
This section is incredibly important for producing high quality user-focused
documentation such as release notes or a development roadmap.  It should be
possible to collect this information before implementation begins in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself.  EEP editors
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this EEP.  Describe why the change is important and the benefits to users.
-->

### Goals

<!--
List the specific goals of the EEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->

### Non-Goals

<!--
What is out of scope for this EEP?  Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that readers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.  The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (optional)

<!--
Detail the things that people will be able to do if this EEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system.  The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable.  This may include API specs (though not always
required) or even code snippets.  If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.  Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The EEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in EEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.


## Implementation History

<!--
Major milestones in the life cycle of a EEP should be tracked in this section.
Major milestones might include
- the date implementation started
- the first EdgeNet release where an initial version of the EEP was available
- the version of EdgeNet where the EEP graduated to general availability
- when the EEP was retired or superseded
-->

## Drawbacks

<!--
Why should this EEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
