package core

import "strings"

type SiteMap struct {
	Pages   []Page
	Library CompositionLibrary
}

type Page struct {
	Key           string
	Label         string
	Route         string
	Group         PageGroup
	GoSXComponent string
	Status        string
	Editable      bool
	Selected      bool
	Components    []Component
}

type PageGroupCount struct {
	Group PageGroup
	Label string
	Count int
}

func (siteMap SiteMap) ComponentCount() int {
	count := 0
	for _, page := range siteMap.Pages {
		count += len(page.Components)
	}
	return count
}

func (siteMap SiteMap) ControlCount() int {
	count := 0
	for _, page := range siteMap.Pages {
		count += page.ControlCount()
	}
	return count
}

func (siteMap SiteMap) Normalize() SiteMap {
	out := SiteMap{
		Pages:   make([]Page, 0, len(siteMap.Pages)),
		Library: siteMap.Library.Normalize(),
	}
	hasSelected := false
	for _, page := range siteMap.Pages {
		page = page.Normalize()
		if page.Key == "" || page.Label == "" || page.Route == "" || page.GoSXComponent == "" {
			continue
		}
		if page.Selected {
			if hasSelected {
				page.Selected = false
			} else {
				hasSelected = true
			}
		}
		out.Pages = append(out.Pages, page)
	}
	if !hasSelected && len(out.Pages) > 0 {
		out.Pages[0].Selected = true
	}
	return out
}

func (siteMap SiteMap) PageGroupCounts() []PageGroupCount {
	counts := map[PageGroup]int{}
	for _, page := range siteMap.Normalize().Pages {
		counts[page.NormalizedGroup()]++
	}

	groups := []PageGroup{
		PageGroupSite,
		PageGroupCommerce,
		PageGroupContent,
		PageGroupFlows,
		PageGroupUtility,
	}
	out := make([]PageGroupCount, 0, len(groups))
	for _, group := range groups {
		count := counts[group]
		if count == 0 {
			continue
		}
		out = append(out, PageGroupCount{
			Group: group,
			Label: PageGroupLabel(group),
			Count: count,
		})
	}
	return out
}

func (siteMap SiteMap) BlueprintCount() int {
	return len(siteMap.Library.PageBlueprints)
}

func (siteMap SiteMap) TemplateCount() int {
	return len(siteMap.Library.ComponentTemplates)
}

func (siteMap SiteMap) SelectedPage() (Page, bool) {
	siteMap = siteMap.Normalize()
	for _, page := range siteMap.Pages {
		if page.Selected {
			return page, true
		}
	}
	if len(siteMap.Pages) > 0 {
		return siteMap.Pages[0], true
	}
	return Page{}, false
}

func (page Page) ComponentCount() int {
	return len(page.Components)
}

func (page Page) Normalize() Page {
	page.Key = strings.TrimSpace(page.Key)
	page.Label = strings.TrimSpace(page.Label)
	page.Route = strings.TrimSpace(page.Route)
	page.Group = normalizePageGroup(page.Group)
	page.GoSXComponent = strings.TrimSpace(page.GoSXComponent)
	page.Status = strings.TrimSpace(page.Status)
	page.Components = normalizeComponents(page.Components)
	return page
}

func (page Page) ControlCount() int {
	count := 0
	for _, component := range page.Components {
		count += component.ControlCount()
	}
	return count
}

func (page Page) GroupLabel() string {
	return PageGroupLabel(page.Group)
}

func (page Page) NormalizedGroup() PageGroup {
	return normalizePageGroup(page.Group)
}

func normalizePageGroup(group PageGroup) PageGroup {
	normalized := PageGroup(strings.TrimSpace(string(group)))
	switch normalized {
	case PageGroupCommerce, PageGroupContent, PageGroupFlows, PageGroupUtility:
		return normalized
	default:
		return PageGroupSite
	}
}
