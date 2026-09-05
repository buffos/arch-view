# OKF knowledge views

Arch View can show two different kinds of read-only information:

- an architecture view built from analyzer model data;
- an OKF knowledge view built from one of the project's independent knowledge bundles.

The architecture viewer remains the first page. When the local server discovers at least one selectable OKF bundle, the architecture header offers **Open OKF knowledge view**. The OKF view can also be opened directly with `?view=okf`. Use `?view=architecture` to return explicitly to the architecture viewer.

If OKF discovery fails, the architecture viewer still starts. OKF is an optional view, not a prerequisite for understanding the architecture model.

## What an OKF view shows

An OKF bundle is a collection of user-authored knowledge documents. Each document is presented as a **concept**. The word is deliberately generic: a concept might describe a capability, decision, workflow, component, person, or any other vocabulary chosen by the bundle author.

The selected profile decides which fields become graph labels, which relationships become containment, which metadata becomes visible, and how states and structure are styled. The viewer does not assume that every concept is a capability or that every bundle has a particular frontmatter field.

The OKF source remains read-only. Arch View indexes and projects the bundle for exploration, but it does not edit the Markdown or frontmatter documents.

## Start an OKF view

Open a project with the local server:

~~~powershell
go run ./cmd/arch-view open --project . --port 0
~~~

Then:

1. Start in the Architecture viewer.
2. Select **Open OKF knowledge view** when the link is available.
3. Choose a bundle and a presentation profile.
4. Adjust depth, layout, or profile settings when needed.

Read [OKF bundles](/formats/okf), [Profiles](/okf/profiles), and [Explore an OKF view](/okf/exploration) for the details.

## What OKF does not do

An OKF view does not convert concepts into architecture-model modules, run an analyzer, or change the project source. Model-only sessions and the current self-contained architecture export do not provide project-backed OKF profile persistence.
