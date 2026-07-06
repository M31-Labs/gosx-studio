package studio

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

//gosx:island
func FlowDesignerIsland(props FlowDesignerProps) gosx.Node {
	selected := signal.New(props.DefaultFlowKey)
	dirtyFlow := signal.New("")
	markDirty := func() { dirtyFlow.Set(selected.Get()) }

	return <section
		class="editor-panel editor-panel--flow-designer studio-flow-designer"
		data-studio-mode-panel="advanced"
		data-studio-panel="flow-designer"
		data-studio-flow-designer-panel="true"
		data-studio-engine-source="gosx"
		data-studio-flow-selected={selected.Get()}
		data-gosx-studio-flow-designer-panel-renderer="gosx-studio"
	>
		<header class="studio-panel-heading">
			<p class="kicker">Advanced</p>
			<h2>Form flows</h2>
			<p>
				Build and publish the site forms people use to contact, request, subscribe, and continue checkout.
			</p>
		</header>
		<div class="studio-flow-designer__grid">
			<div class="studio-flow-designer__list" role="list" aria-label="Form flows">
				<Each of={props.Flows} as="flow">
					<button
						type="button"
						class={flow.CardClass}
						onClick={func() { selected.Set(flow.Key) }}
						data-editor-flow={flow.Key}
						data-studio-flow-card={flow.Key}
						data-studio-flow-card-selected={selected.Get() == flow.Key}
						data-studio-flow-route={flow.Route}
						data-studio-flow-label={flow.Label}
						data-studio-flow-status={flow.StatusLabel}
						aria-pressed={selected.Get() == flow.Key}
					>
						<span class="studio-flow-card__head">
							<strong>{flow.Label}</strong>
							<output>{flow.StatusLabel}</output>
						</span>
						<span class="studio-flow-card__body">{flow.Description}</span>
						<span class="studio-flow-card__meta">
							<span>{flow.Summary}</span>
							<span>{flow.HandlerStatusLabel}</span>
						</span>
						<output
							class="studio-flow-summary__dirty"
							data-studio-flow-dirty-badge={flow.Key}
							data-studio-flow-dirty-visible={dirtyFlow.Get() == flow.Key}
						>Unsaved</output>
					</button>
				</Each>
			</div>
			<div class="studio-flow-designer__editors">
				<Each of={props.Flows} as="flow">
					<article
						class="studio-flow-editor"
						data-editor-flow={flow.Key}
						data-studio-flow-editor={flow.Key}
						data-studio-flow-valid={flow.Readiness.CanPublish}
						data-studio-flow-readiness-status={flow.Readiness.Status}
						data-studio-flow-dirty={dirtyFlow.Get() == flow.Key}
						data-studio-flow-editor-visible={selected.Get() == flow.Key}
					>
						<header class="studio-flow-editor__head">
							<div>
								<p class="kicker">{flow.StatusLabel}</p>
								<h3>{flow.Label}</h3>
								<p>{flow.Summary}</p>
							</div>
							<output
								class={flow.Readiness.BadgeClass}
								data-studio-flow-editor-state={flow.Key}
								data-studio-flow-editor-state-visible={dirtyFlow.Get() != flow.Key}
							>{flow.Readiness.Label}</output>
							<output
								class="studio-flow-editor__state studio-flow-editor__state--watch"
								data-studio-flow-editor-state={flow.Key}
								data-studio-flow-editor-state-visible={dirtyFlow.Get() == flow.Key}
							>Unsaved changes</output>
							<button
								class="button button--secondary"
								type="submit"
								form={props.FormID}
								formaction={props.PublishAction}
								formmethod="post"
								name="flowKey"
								value={flow.Key}
								data-studio-submit-action="publish-flow"
							>Publish flow</button>
						</header>
						<div class={flow.Readiness.Class} data-studio-flow-readiness={flow.Key}>
							<header>
								<strong>{flow.Readiness.Label}</strong>
								<span>{flow.Readiness.Summary}</span>
							</header>
							<div class="studio-flow-checks" role="list" aria-label="Flow readiness checks">
								<Each of={flow.ReadinessChecks} as="check">
									<section
										class={check.Class}
										role="listitem"
										tabindex={check.TabIndex}
										data-studio-flow-check={check.Key}
										data-studio-flow-check-status={check.Status}
									>
										<span>{check.StatusLabel}</span>
										<strong>{check.Label}</strong>
										<small>{check.Summary}</small>
									</section>
								</Each>
							</div>
						</div>
						<div class="studio-flow-graph" role="list" aria-label="Flow map">
							<Each of={flow.Nodes} as="node">
								<section
									class={node.Class}
									role="listitem"
									tabindex={node.TabIndex}
									data-studio-flow-node={node.Key}
									data-studio-flow-node-kind={node.Kind}
									data-studio-flow-node-status={node.Status}
								>
									<span>{node.KindLabel}</span>
									<strong>{node.Label}</strong>
									<small>{node.Summary}</small>
								</section>
							</Each>
						</div>
						<div class="studio-flow-editor__route">
							<label class="field" for={flow.RouteInputID}>
								<span>Public page</span>
								<input
									id={flow.RouteInputID}
									name={flow.RouteInputName}
									value={flow.Route}
									form={props.FormID}
									data-studio-field-source={"flow." + flow.Key + ".route"}
									data-studio-field-editable="routing"
									readonly
								 />
							</label>
							<label class="field" for={flow.EmbedInputID}>
								<span>Appears in</span>
								<input
									id={flow.EmbedInputID}
									name={flow.EmbedInputName}
									value={flow.EmbedTarget}
									form={props.FormID}
									data-studio-field-source={"flow." + flow.Key + ".embedTarget"}
									data-studio-field-editable="routing"
									readonly
								 />
							</label>
							<input
								id={flow.HandlerInputID}
								name={flow.HandlerInputName}
								value={flow.HandlerRef}
								form={props.FormID}
								data-studio-field-source={flow.HandlerSource}
								data-studio-field-editable="flow"
								data-studio-flow-handler-ref="true"
								type="hidden"
							 />
							<div class={flow.HandlerClass} data-studio-flow-handler-summary={flow.Key}>
								<strong>{flow.HandlerStatusLabel}</strong>
								<span>{flow.HandlerDetail}</span>
							</div>
						</div>
						<div class="studio-flow-editor__steps" aria-label="Flow steps">
							<Each of={flow.Steps} as="step">
								<section class="studio-flow-step" data-studio-flow-step={step.Key} tabindex="0">
									<label class="field" for={step.LabelInputID}>
										<span>{step.Label}</span>
										<input
											id={step.LabelInputID}
											name={step.LabelInputName}
											value={step.Label}
											form={props.FormID}
											data-studio-field-source={step.LabelSource}
											data-studio-field-editable="flow"
											onInput={markDirty}
											onChange={markDirty}
										 />
									</label>
									<p
										class="studio-flow-step__body"
										data-studio-field-source={step.BodySource}
										data-studio-field-editable="flow"
									>{step.BodySummary}</p>
								</section>
							</Each>
						</div>
						<div class="studio-flow-editor__actions" aria-label="Flow actions">
							<Each of={flow.Actions} as="action">
								<section class="studio-flow-action" data-studio-flow-action={action.Key} tabindex="0">
									<header>
										<strong>{action.Label}</strong>
										<span>
											{action.FieldCount}
											fields
										</span>
									</header>
									<div class="studio-flow-field-list">
										<Each of={action.Fields} as="field">
											<span class="studio-flow-field" data-studio-flow-field={field.Name} tabindex="0">
												<strong>{field.Label}</strong>
												<small>{field.RequiredLabel}</small>
											</span>
										</Each>
									</div>
								</section>
							</Each>
						</div>
					</article>
				</Each>
			</div>
		</div>
	</section>
}
