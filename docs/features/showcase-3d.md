# Showcase 3D

Showcase 3D covers the feature request for AI-generated 3D assets from photos of physical pieces, with inline and pop-out rendering for public pages.

This should live as a CMS showcase plugin plus Studio authoring controls, not as one-off Noni-site code.

## Responsibilities

CMS owns:

- piece asset records
- source photo sets
- generated model metadata
- provenance and moderation state
- lifecycle fields for review, approval, and replacement

Studio owns:

- authoring UI for choosing or generating a model
- placement controls for pages and sections
- preview state
- validation before publish
- plain-language controls for non-technical editors

GoSX Scene3D or a Studio engine owns:

- model loading
- camera and controls
- inline preview
- public pop-out viewing
- fallback poster rendering

The host app owns:

- provider credentials
- product labels
- page placement rules
- public route decisions

## Target Flow

1. Editor uploads or selects photos for a piece.
2. CMS records a generation request and generated model metadata.
3. Editor reviews the generated model in Studio.
4. Editor places the model in a showcase section.
5. Public site renders an inline viewer with an optional pop-out viewer.

## Open Decisions

- Use GLB as the default generated model artifact unless a provider requires another intermediate format.
- Define the generation provider as an adapter interface.
- Store provenance and moderation metadata before publishing generated models.
- Provide a poster image and lazy-loading fallback for slow devices.
