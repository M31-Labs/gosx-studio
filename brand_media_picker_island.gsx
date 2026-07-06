package studio

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

//gosx:island
func BrandMediaPickerIsland(props BrandMediaPickerProps) gosx.Node {
	filter := signal.New("all")
	showAll := func() { filter.Set("all") }
	showReady := func() { filter.Set("ready") }
	showMissingAlt := func() { filter.Set("missing-alt") }

	return <section
		class="studio-brand-media-picker"
		data-studio-media-picker-island="brand"
		data-studio-engine-source="gosx"
		data-studio-media-filter={filter.Get()}
		data-gosx-studio-brand-media-picker-renderer="gosx-studio"
	>
		<header class="studio-brand-media-picker__head">
			<div>
				<p class="kicker">{props.Kicker}</p>
				<h3>{props.Title}</h3>
				<p>{props.Summary}</p>
			</div>
			<a href={props.UploadHref} data-gosx-link="true">Upload</a>
		</header>
		<div class="studio-brand-media-picker__filters" role="toolbar" aria-label={props.FilterLabel}>
			<button type="button" onClick={showAll} aria-pressed={filter.Get() == "all"}>All</button>
			<button type="button" onClick={showReady} aria-pressed={filter.Get() == "ready"}>Ready</button>
			<button type="button" onClick={showMissingAlt} aria-pressed={filter.Get() == "missing-alt"}>Needs alt</button>
		</div>
		<p class="studio-brand-media-picker__empty" hidden={props.HasAssets}>{props.EmptyLabel}</p>
		<div class="studio-brand-media-picker__grid" aria-label={props.AssetLabel}>
			<Each of={props.Assets} as="asset">
				<article
					class={asset.CardClass}
					data-studio-media-asset={asset.ID}
					hidden={filter.Get() != "all" && asset.FilterGroup != filter.Get()}
				>
					<img src={asset.URL} alt={asset.Alt} />
					<div>
						<strong>{asset.Filename}</strong>
						<span>{asset.StatusLabel}</span>
					</div>
					<div class="studio-brand-media-picker__actions" aria-label={asset.ActionLabel}>
						<button
							type="submit"
							form={props.FormID}
							formaction={props.SaveAction}
							name={props.LogoPickName}
							value={asset.URL}
						>Use logo</button>
						<button
							type="submit"
							form={props.FormID}
							formaction={props.SaveAction}
							name={props.FaviconPickName}
							value={asset.URL}
						>Use icon</button>
					</div>
				</article>
			</Each>
		</div>
	</section>
}
