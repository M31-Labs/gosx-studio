package sitemap

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

//gosx:island
func SiteNavigatorIsland(props SiteNavigatorProps) gosx.Node {
	filter := signal.New("all")
	showAll := func() { filter.Set("all") }
	showSite := func() { filter.Set("site") }
	showContent := func() { filter.Set("content") }
	showStore := func() { filter.Set("store") }
	showTools := func() { filter.Set("tools") }

	return <section
		class="studio-nav-panel"
		data-studio-site-navigator-panel="true"
		data-studio-mode-panel={props.Mode}
		data-studio-engine-source="gosx"
		data-studio-site-filter={filter.Get()}
		data-gosx-studio-site-navigator-renderer="gosx-studio"
	>
		<div class="studio-panel-heading">
			<p class="kicker">{props.Kicker}</p>
			<h2>{props.Title}</h2>
		</div>
		<div class="studio-page-filter" role="toolbar" aria-label="Site area filter">
			<button type="button" onClick={showAll} aria-pressed={filter.Get() == "all"}>All</button>
			<button type="button" onClick={showSite} aria-pressed={filter.Get() == "site"}>Site</button>
			<button type="button" onClick={showContent} aria-pressed={filter.Get() == "content"}>Content</button>
			<button type="button" onClick={showStore} aria-pressed={filter.Get() == "store"}>Store</button>
			<button type="button" onClick={showTools} aria-pressed={filter.Get() == "tools"}>Tools</button>
		</div>
		<nav class="studio-page-list" aria-label={props.Label}>
			<p class="empty" hidden={props.HasItems}>{props.Empty}</p>
			<Each of={props.Items} as="item">
				<a
					href={item.Href}
					class={item.Class}
					data-gosx-link="true"
					data-studio-site-page={item.Key}
					data-studio-site-group={item.Group}
					title={item.Summary}
					hidden={filter.Get() != "all" && item.Group != filter.Get()}
				>
					<span>{item.Label}</span>
					<span class="studio-page-list__summary" hidden={item.Summary == ""}>{item.Summary}</span>
				</a>
			</Each>
		</nav>
	</section>
}
