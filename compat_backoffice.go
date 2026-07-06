package studio

// This file is the deprecated compatibility facade for the symbols that used
// to live in the root package's 21 backend_*.go CRUD/back-office files
// (backend_blog_{index,detail}.go, backend_category_{index,detail}.go,
// backend_contact_detail.go, backend_dashboard.go, backend_detail.go,
// backend_gallery_{index,detail}.go, backend_media_{library,detail}.go,
// backend_order_detail.go, backend_page_{index,detail}.go,
// backend_product_{index,detail}.go, backend_resource_index.go,
// backend_search.go, backend_settings.go, backend_storefront_preview.go,
// and backend_workbench.go) before Slice 7 of the package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) extracted them into
// the m31labs.dev/gosx-studio/backoffice package: the CRUD portal surfaces —
// dashboard, per-domain index/detail pages, media library, settings,
// search, storefront preview, generic resource table/detail scaffolding.
//
// backend_editor_page.go and backend_editor_workbench.go (plus their tests)
// are explicitly OUT of scope for this slice: they are shell composition
// territory (Slice 8), never backoffice, per the spec's DAG rule that
// backoffice and shell never import each other. They remain at root and
// call no symbol from this facade (verified: neither editor file nor its
// test references any Backend{Blog,Category,Contact,Dashboard,Detail,
// Gallery,Media,Order,Page,Product,Resource,Search,Settings,
// StorefrontPreview,Workbench}* symbol).
//
// Every exported type below is a type ALIAS (not a new type), so struct
// literals, embedding, and assignability are all unchanged for existing
// callers (muddy-noni-commerce, pajaritos-forest-school, gosx-cms/studio).
// Every exported free function is a thin forwarding wrapper. Every rendered
// data-studio-*/data-gosx-studio-* attribute name and value is part of the
// rendered-page contract and must not change (spec risk R5).
//
// backoffice imports only core (per the spec's §1 Import DAG) — in
// practice the 21 moved files import nothing beyond m31labs.dev/gosx, so
// there are no frozen-copy helpers or cross-package shims needed for this
// slice (unlike Slices 4-6): the moved production and test code is fully
// self-contained.
//
// Deprecated: import m31labs.dev/gosx-studio/backoffice directly; this
// facade will be removed after the v0.6.x compatibility window (see spec
// §9).

import (
	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/backoffice"
)

// --- backend_blog_detail.go ---

// Deprecated: use backoffice.BackendBlogDetailPageProps.
type BackendBlogDetailPageProps = backoffice.BackendBlogDetailPageProps

// Deprecated: use backoffice.BackendBlogDetailActionStatus.
type BackendBlogDetailActionStatus = backoffice.BackendBlogDetailActionStatus

// Deprecated: use backoffice.BackendBlogDetailMediaAsset.
type BackendBlogDetailMediaAsset = backoffice.BackendBlogDetailMediaAsset

// Deprecated: use backoffice.BackendBlogDetailTag.
type BackendBlogDetailTag = backoffice.BackendBlogDetailTag

// Deprecated: use backoffice.BackendBlogDetailRelation.
type BackendBlogDetailRelation = backoffice.BackendBlogDetailRelation

// Deprecated: use backoffice.BackendBlogDetailValues.
type BackendBlogDetailValues = backoffice.BackendBlogDetailValues

// Deprecated: use backoffice.RenderBackendBlogDetailPage.
func RenderBackendBlogDetailPage(props BackendBlogDetailPageProps) gosx.Node {
	return backoffice.RenderBackendBlogDetailPage(props)
}

// Deprecated: use backoffice.RenderBackendBlogDetailContent.
func RenderBackendBlogDetailContent(props BackendBlogDetailPageProps) gosx.Node {
	return backoffice.RenderBackendBlogDetailContent(props)
}

// Deprecated: use backoffice.RenderBackendBlogDetailMediaDatalist.
func RenderBackendBlogDetailMediaDatalist(media []BackendBlogDetailMediaAsset) gosx.Node {
	return backoffice.RenderBackendBlogDetailMediaDatalist(media)
}

// Deprecated: use backoffice.RenderBackendBlogDetailTagsDatalist.
func RenderBackendBlogDetailTagsDatalist(tags []BackendBlogDetailTag) gosx.Node {
	return backoffice.RenderBackendBlogDetailTagsDatalist(tags)
}

// Deprecated: use backoffice.RenderBackendBlogDetailForm.
func RenderBackendBlogDetailForm(props BackendBlogDetailPageProps) gosx.Node {
	return backoffice.RenderBackendBlogDetailForm(props)
}

// Deprecated: use backoffice.RenderBackendBlogDetailScripts.
func RenderBackendBlogDetailScripts() gosx.Node {
	return backoffice.RenderBackendBlogDetailScripts()
}

// --- backend_blog_index.go ---

// Deprecated: use backoffice.BackendBlogIndexPageProps.
type BackendBlogIndexPageProps = backoffice.BackendBlogIndexPageProps

// Deprecated: use backoffice.BackendBlogIndexActionStatus.
type BackendBlogIndexActionStatus = backoffice.BackendBlogIndexActionStatus

// Deprecated: use backoffice.BackendBlogIndexMediaAsset.
type BackendBlogIndexMediaAsset = backoffice.BackendBlogIndexMediaAsset

// Deprecated: use backoffice.BackendBlogIndexTag.
type BackendBlogIndexTag = backoffice.BackendBlogIndexTag

// Deprecated: use backoffice.BackendBlogIndexRelation.
type BackendBlogIndexRelation = backoffice.BackendBlogIndexRelation

// Deprecated: use backoffice.BackendBlogIndexValues.
type BackendBlogIndexValues = backoffice.BackendBlogIndexValues

// Deprecated: use backoffice.RenderBackendBlogIndexPage.
func RenderBackendBlogIndexPage(props BackendBlogIndexPageProps) gosx.Node {
	return backoffice.RenderBackendBlogIndexPage(props)
}

// Deprecated: use backoffice.RenderBackendBlogIndexContent.
func RenderBackendBlogIndexContent(props BackendBlogIndexPageProps) gosx.Node {
	return backoffice.RenderBackendBlogIndexContent(props)
}

// Deprecated: use backoffice.RenderBackendBlogCreatePanel.
func RenderBackendBlogCreatePanel(props BackendBlogIndexPageProps) gosx.Node {
	return backoffice.RenderBackendBlogCreatePanel(props)
}

// Deprecated: use backoffice.RenderBackendBlogIndexMediaDatalist.
func RenderBackendBlogIndexMediaDatalist(media []BackendBlogIndexMediaAsset) gosx.Node {
	return backoffice.RenderBackendBlogIndexMediaDatalist(media)
}

// Deprecated: use backoffice.RenderBackendBlogIndexTagsDatalist.
func RenderBackendBlogIndexTagsDatalist(tags []BackendBlogIndexTag) gosx.Node {
	return backoffice.RenderBackendBlogIndexTagsDatalist(tags)
}

// Deprecated: use backoffice.RenderBackendBlogIndexScripts.
func RenderBackendBlogIndexScripts() gosx.Node {
	return backoffice.RenderBackendBlogIndexScripts()
}

// --- backend_category_detail.go ---

// Deprecated: use backoffice.BackendCategoryDetailPageProps.
type BackendCategoryDetailPageProps = backoffice.BackendCategoryDetailPageProps

// Deprecated: use backoffice.BackendCategoryDetailActionStatus.
type BackendCategoryDetailActionStatus = backoffice.BackendCategoryDetailActionStatus

// Deprecated: use backoffice.BackendCategoryDetailValues.
type BackendCategoryDetailValues = backoffice.BackendCategoryDetailValues

// Deprecated: use backoffice.RenderBackendCategoryDetailPage.
func RenderBackendCategoryDetailPage(props BackendCategoryDetailPageProps) gosx.Node {
	return backoffice.RenderBackendCategoryDetailPage(props)
}

// Deprecated: use backoffice.RenderBackendCategoryDetailContent.
func RenderBackendCategoryDetailContent(props BackendCategoryDetailPageProps) gosx.Node {
	return backoffice.RenderBackendCategoryDetailContent(props)
}

// Deprecated: use backoffice.RenderBackendCategoryDetailForm.
func RenderBackendCategoryDetailForm(props BackendCategoryDetailPageProps) gosx.Node {
	return backoffice.RenderBackendCategoryDetailForm(props)
}

// --- backend_category_index.go ---

// Deprecated: use backoffice.BackendCategoryIndexPageProps.
type BackendCategoryIndexPageProps = backoffice.BackendCategoryIndexPageProps

// Deprecated: use backoffice.BackendCategoryIndexActionStatus.
type BackendCategoryIndexActionStatus = backoffice.BackendCategoryIndexActionStatus

// Deprecated: use backoffice.BackendCategoryIndexValues.
type BackendCategoryIndexValues = backoffice.BackendCategoryIndexValues

// Deprecated: use backoffice.RenderBackendCategoryIndexPage.
func RenderBackendCategoryIndexPage(props BackendCategoryIndexPageProps) gosx.Node {
	return backoffice.RenderBackendCategoryIndexPage(props)
}

// Deprecated: use backoffice.RenderBackendCategoryIndexContent.
func RenderBackendCategoryIndexContent(props BackendCategoryIndexPageProps) gosx.Node {
	return backoffice.RenderBackendCategoryIndexContent(props)
}

// Deprecated: use backoffice.RenderBackendCategoryCreatePanel.
func RenderBackendCategoryCreatePanel(props BackendCategoryIndexPageProps) gosx.Node {
	return backoffice.RenderBackendCategoryCreatePanel(props)
}

// --- backend_contact_detail.go ---

// Deprecated: use backoffice.BackendContactDetailProps.
type BackendContactDetailProps = backoffice.BackendContactDetailProps

// Deprecated: use backoffice.BackendContactMessage.
type BackendContactMessage = backoffice.BackendContactMessage

// Deprecated: use backoffice.BackendContactReply.
type BackendContactReply = backoffice.BackendContactReply

// Deprecated: use backoffice.BackendContactSubmission.
type BackendContactSubmission = backoffice.BackendContactSubmission

// Deprecated: use backoffice.BackendContactActions.
type BackendContactActions = backoffice.BackendContactActions

// Deprecated: use backoffice.BackendContactActionPaths.
type BackendContactActionPaths = backoffice.BackendContactActionPaths

// Deprecated: use backoffice.BackendContactReplyActionState.
type BackendContactReplyActionState = backoffice.BackendContactReplyActionState

// Deprecated: use backoffice.BackendContactTime.
type BackendContactTime = backoffice.BackendContactTime

// Deprecated: use backoffice.RenderBackendContactDetail.
func RenderBackendContactDetail(props BackendContactDetailProps) gosx.Node {
	return backoffice.RenderBackendContactDetail(props)
}

// Deprecated: use backoffice.RenderBackendContactDetailContent.
func RenderBackendContactDetailContent(props BackendContactDetailProps) gosx.Node {
	return backoffice.RenderBackendContactDetailContent(props)
}

// Deprecated: use backoffice.RenderBackendContactDetailPage.
func RenderBackendContactDetailPage(props BackendContactDetailProps) gosx.Node {
	return backoffice.RenderBackendContactDetailPage(props)
}

// Deprecated: use backoffice.RenderBackendContactDetailPageContent.
func RenderBackendContactDetailPageContent(props BackendContactDetailProps) gosx.Node {
	return backoffice.RenderBackendContactDetailPageContent(props)
}

// Deprecated: use backoffice.RenderBackendContactDetailHeading.
func RenderBackendContactDetailHeading(props BackendContactDetailProps) gosx.Node {
	return backoffice.RenderBackendContactDetailHeading(props)
}

// Deprecated: use backoffice.RenderBackendContactMessage.
func RenderBackendContactMessage(message BackendContactMessage) gosx.Node {
	return backoffice.RenderBackendContactMessage(message)
}

// Deprecated: use backoffice.RenderBackendContactStatusForm.
func RenderBackendContactStatusForm(actions BackendContactActions) gosx.Node {
	return backoffice.RenderBackendContactStatusForm(actions)
}

// Deprecated: use backoffice.RenderBackendContactReplyForm.
func RenderBackendContactReplyForm(actions BackendContactActions) gosx.Node {
	return backoffice.RenderBackendContactReplyForm(actions)
}

// Deprecated: use backoffice.RenderBackendContactReplyHistory.
func RenderBackendContactReplyHistory(replies []BackendContactReply) gosx.Node {
	return backoffice.RenderBackendContactReplyHistory(replies)
}

// Deprecated: use backoffice.RenderBackendContactSubmission.
func RenderBackendContactSubmission(submission BackendContactSubmission) gosx.Node {
	return backoffice.RenderBackendContactSubmission(submission)
}

// --- backend_dashboard.go ---

// Deprecated: use backoffice.BackendDashboardProps.
type BackendDashboardProps = backoffice.BackendDashboardProps

// Deprecated: use backoffice.BackendDashboardStat.
type BackendDashboardStat = backoffice.BackendDashboardStat

// Deprecated: use backoffice.BackendDashboardCard.
type BackendDashboardCard = backoffice.BackendDashboardCard

// Deprecated: use backoffice.BackendDashboardPayment.
type BackendDashboardPayment = backoffice.BackendDashboardPayment

// Deprecated: use backoffice.BackendDashboardActionState.
type BackendDashboardActionState = backoffice.BackendDashboardActionState

// Deprecated: use backoffice.BackendDashboardTime.
type BackendDashboardTime = backoffice.BackendDashboardTime

// Deprecated: use backoffice.BackendDashboardAlert.
type BackendDashboardAlert = backoffice.BackendDashboardAlert

// Deprecated: use backoffice.BackendDashboardChecklistItem.
type BackendDashboardChecklistItem = backoffice.BackendDashboardChecklistItem

// Deprecated: use backoffice.BackendDashboardResource.
type BackendDashboardResource = backoffice.BackendDashboardResource

// Deprecated: use backoffice.BackendDashboardTimelineEvent.
type BackendDashboardTimelineEvent = backoffice.BackendDashboardTimelineEvent

// Deprecated: use backoffice.BackendDashboardAuth.
type BackendDashboardAuth = backoffice.BackendDashboardAuth

// Deprecated: use backoffice.BackendDashboardOrder.
type BackendDashboardOrder = backoffice.BackendDashboardOrder

// Deprecated: use backoffice.BackendDashboardContact.
type BackendDashboardContact = backoffice.BackendDashboardContact

// Deprecated: use backoffice.RenderBackendDashboard.
func RenderBackendDashboard(props BackendDashboardProps) gosx.Node {
	return backoffice.RenderBackendDashboard(props)
}

// --- backend_detail.go ---

// Deprecated: use backoffice.BackendDetailProps.
type BackendDetailProps = backoffice.BackendDetailProps

// Deprecated: use backoffice.BackendDetailPreview.
type BackendDetailPreview = backoffice.BackendDetailPreview

// Deprecated: use backoffice.BackendDetailPreviewImage.
type BackendDetailPreviewImage = backoffice.BackendDetailPreviewImage

// Deprecated: use backoffice.BackendDetailStatus.
type BackendDetailStatus = backoffice.BackendDetailStatus

// Deprecated: use backoffice.BackendDetailTime.
type BackendDetailTime = backoffice.BackendDetailTime

// Deprecated: use backoffice.RenderBackendDetail.
func RenderBackendDetail(props BackendDetailProps) gosx.Node {
	return backoffice.RenderBackendDetail(props)
}

// Deprecated: use backoffice.RenderBackendDetailContent.
func RenderBackendDetailContent(props BackendDetailProps) gosx.Node {
	return backoffice.RenderBackendDetailContent(props)
}

// Deprecated: use backoffice.RenderBackendDetailHeading.
func RenderBackendDetailHeading(props BackendDetailProps) gosx.Node {
	return backoffice.RenderBackendDetailHeading(props)
}

// Deprecated: use backoffice.RenderBackendDetailPreview.
func RenderBackendDetailPreview(preview BackendDetailPreview) gosx.Node {
	return backoffice.RenderBackendDetailPreview(preview)
}

// --- backend_gallery_detail.go ---

// Deprecated: use backoffice.BackendGalleryDetailPageProps.
type BackendGalleryDetailPageProps = backoffice.BackendGalleryDetailPageProps

// Deprecated: use backoffice.BackendGalleryDetailActionStatus.
type BackendGalleryDetailActionStatus = backoffice.BackendGalleryDetailActionStatus

// Deprecated: use backoffice.BackendGalleryDetailMediaAsset.
type BackendGalleryDetailMediaAsset = backoffice.BackendGalleryDetailMediaAsset

// Deprecated: use backoffice.BackendGalleryDetailValues.
type BackendGalleryDetailValues = backoffice.BackendGalleryDetailValues

// Deprecated: use backoffice.BackendGalleryDetailImage.
type BackendGalleryDetailImage = backoffice.BackendGalleryDetailImage

// Deprecated: use backoffice.RenderBackendGalleryDetailPage.
func RenderBackendGalleryDetailPage(props BackendGalleryDetailPageProps) gosx.Node {
	return backoffice.RenderBackendGalleryDetailPage(props)
}

// Deprecated: use backoffice.RenderBackendGalleryDetailContent.
func RenderBackendGalleryDetailContent(props BackendGalleryDetailPageProps) gosx.Node {
	return backoffice.RenderBackendGalleryDetailContent(props)
}

// Deprecated: use backoffice.RenderBackendGalleryDetailMediaDatalist.
func RenderBackendGalleryDetailMediaDatalist(media []BackendGalleryDetailMediaAsset) gosx.Node {
	return backoffice.RenderBackendGalleryDetailMediaDatalist(media)
}

// Deprecated: use backoffice.RenderBackendGalleryDetailForm.
func RenderBackendGalleryDetailForm(props BackendGalleryDetailPageProps) gosx.Node {
	return backoffice.RenderBackendGalleryDetailForm(props)
}

// Deprecated: use backoffice.RenderBackendGalleryDetailScripts.
func RenderBackendGalleryDetailScripts() gosx.Node {
	return backoffice.RenderBackendGalleryDetailScripts()
}

// --- backend_gallery_index.go ---

// Deprecated: use backoffice.BackendGalleryIndexPageProps.
type BackendGalleryIndexPageProps = backoffice.BackendGalleryIndexPageProps

// Deprecated: use backoffice.BackendGalleryIndexActionStatus.
type BackendGalleryIndexActionStatus = backoffice.BackendGalleryIndexActionStatus

// Deprecated: use backoffice.BackendGalleryIndexMediaAsset.
type BackendGalleryIndexMediaAsset = backoffice.BackendGalleryIndexMediaAsset

// Deprecated: use backoffice.BackendGalleryIndexValues.
type BackendGalleryIndexValues = backoffice.BackendGalleryIndexValues

// Deprecated: use backoffice.RenderBackendGalleryIndexPage.
func RenderBackendGalleryIndexPage(props BackendGalleryIndexPageProps) gosx.Node {
	return backoffice.RenderBackendGalleryIndexPage(props)
}

// Deprecated: use backoffice.RenderBackendGalleryIndexContent.
func RenderBackendGalleryIndexContent(props BackendGalleryIndexPageProps) gosx.Node {
	return backoffice.RenderBackendGalleryIndexContent(props)
}

// Deprecated: use backoffice.RenderBackendGalleryCreatePanel.
func RenderBackendGalleryCreatePanel(props BackendGalleryIndexPageProps) gosx.Node {
	return backoffice.RenderBackendGalleryCreatePanel(props)
}

// Deprecated: use backoffice.RenderBackendGalleryIndexMediaDatalist.
func RenderBackendGalleryIndexMediaDatalist(media []BackendGalleryIndexMediaAsset) gosx.Node {
	return backoffice.RenderBackendGalleryIndexMediaDatalist(media)
}

// Deprecated: use backoffice.RenderBackendGalleryIndexScripts.
func RenderBackendGalleryIndexScripts() gosx.Node {
	return backoffice.RenderBackendGalleryIndexScripts()
}

// --- backend_media_detail.go ---

// Deprecated: use backoffice.BackendMediaDetailPageProps.
type BackendMediaDetailPageProps = backoffice.BackendMediaDetailPageProps

// Deprecated: use backoffice.BackendMediaDetailActionStatus.
type BackendMediaDetailActionStatus = backoffice.BackendMediaDetailActionStatus

// Deprecated: use backoffice.BackendMediaDetailAsset.
type BackendMediaDetailAsset = backoffice.BackendMediaDetailAsset

// Deprecated: use backoffice.BackendMediaDetailVariant.
type BackendMediaDetailVariant = backoffice.BackendMediaDetailVariant

// Deprecated: use backoffice.BackendMediaDetailUsage.
type BackendMediaDetailUsage = backoffice.BackendMediaDetailUsage

// Deprecated: use backoffice.RenderBackendMediaDetailPage.
func RenderBackendMediaDetailPage(props BackendMediaDetailPageProps) gosx.Node {
	return backoffice.RenderBackendMediaDetailPage(props)
}

// Deprecated: use backoffice.RenderBackendMediaDetailContent.
func RenderBackendMediaDetailContent(props BackendMediaDetailPageProps) gosx.Node {
	return backoffice.RenderBackendMediaDetailContent(props)
}

// Deprecated: use backoffice.RenderBackendMediaDetailForm.
func RenderBackendMediaDetailForm(props BackendMediaDetailPageProps) gosx.Node {
	return backoffice.RenderBackendMediaDetailForm(props)
}

// Deprecated: use backoffice.RenderBackendMediaReplaceForm.
func RenderBackendMediaReplaceForm(props BackendMediaDetailPageProps) gosx.Node {
	return backoffice.RenderBackendMediaReplaceForm(props)
}

// --- backend_media_library.go ---

// Deprecated: use backoffice.BackendMediaLibraryProps.
type BackendMediaLibraryProps = backoffice.BackendMediaLibraryProps

// Deprecated: use backoffice.BackendMediaAsset.
type BackendMediaAsset = backoffice.BackendMediaAsset

// Deprecated: use backoffice.BackendMediaLibraryPageProps.
type BackendMediaLibraryPageProps = backoffice.BackendMediaLibraryPageProps

// Deprecated: use backoffice.BackendMediaLibraryActionStatus.
type BackendMediaLibraryActionStatus = backoffice.BackendMediaLibraryActionStatus

// Deprecated: use backoffice.BackendMediaLibraryRemoteAssetValues.
type BackendMediaLibraryRemoteAssetValues = backoffice.BackendMediaLibraryRemoteAssetValues

// Deprecated: use backoffice.RenderBackendMediaLibrary.
func RenderBackendMediaLibrary(props BackendMediaLibraryProps) gosx.Node {
	return backoffice.RenderBackendMediaLibrary(props)
}

// Deprecated: use backoffice.RenderBackendMediaLibraryPage.
func RenderBackendMediaLibraryPage(props BackendMediaLibraryPageProps) gosx.Node {
	return backoffice.RenderBackendMediaLibraryPage(props)
}

// Deprecated: use backoffice.RenderBackendMediaLibraryPageContent.
func RenderBackendMediaLibraryPageContent(props BackendMediaLibraryPageProps) gosx.Node {
	return backoffice.RenderBackendMediaLibraryPageContent(props)
}

// Deprecated: use backoffice.RenderBackendMediaLibraryActions.
func RenderBackendMediaLibraryActions(props BackendMediaLibraryPageProps) gosx.Node {
	return backoffice.RenderBackendMediaLibraryActions(props)
}

// Deprecated: use backoffice.RenderBackendMediaLibraryUploadForm.
func RenderBackendMediaLibraryUploadForm(props BackendMediaLibraryPageProps) gosx.Node {
	return backoffice.RenderBackendMediaLibraryUploadForm(props)
}

// Deprecated: use backoffice.RenderBackendMediaLibraryRemoteAssetForm.
func RenderBackendMediaLibraryRemoteAssetForm(props BackendMediaLibraryPageProps) gosx.Node {
	return backoffice.RenderBackendMediaLibraryRemoteAssetForm(props)
}

// Deprecated: use backoffice.RenderBackendMediaLibraryContent.
func RenderBackendMediaLibraryContent(props BackendMediaLibraryProps) gosx.Node {
	return backoffice.RenderBackendMediaLibraryContent(props)
}

// Deprecated: use backoffice.RenderBackendMediaLibraryHeading.
func RenderBackendMediaLibraryHeading(props BackendMediaLibraryProps) gosx.Node {
	return backoffice.RenderBackendMediaLibraryHeading(props)
}

// Deprecated: use backoffice.RenderBackendMediaLibraryAssets.
func RenderBackendMediaLibraryAssets(props BackendMediaLibraryProps) gosx.Node {
	return backoffice.RenderBackendMediaLibraryAssets(props)
}

// --- backend_order_detail.go ---

// Deprecated: use backoffice.BackendOrderDetailProps.
type BackendOrderDetailProps = backoffice.BackendOrderDetailProps

// Deprecated: use backoffice.BackendOrderSummary.
type BackendOrderSummary = backoffice.BackendOrderSummary

// Deprecated: use backoffice.BackendOrderItem.
type BackendOrderItem = backoffice.BackendOrderItem

// Deprecated: use backoffice.BackendOrderActions.
type BackendOrderActions = backoffice.BackendOrderActions

// Deprecated: use backoffice.BackendOrderActionPaths.
type BackendOrderActionPaths = backoffice.BackendOrderActionPaths

// Deprecated: use backoffice.BackendOrderFulfillmentOption.
type BackendOrderFulfillmentOption = backoffice.BackendOrderFulfillmentOption

// Deprecated: use backoffice.BackendOrderNotes.
type BackendOrderNotes = backoffice.BackendOrderNotes

// Deprecated: use backoffice.BackendOrderSpecRow.
type BackendOrderSpecRow = backoffice.BackendOrderSpecRow

// Deprecated: use backoffice.BackendOrderPaymentReferences.
type BackendOrderPaymentReferences = backoffice.BackendOrderPaymentReferences

// Deprecated: use backoffice.BackendOrderTimelineEvent.
type BackendOrderTimelineEvent = backoffice.BackendOrderTimelineEvent

// Deprecated: use backoffice.BackendOrderAuditEvent.
type BackendOrderAuditEvent = backoffice.BackendOrderAuditEvent

// Deprecated: use backoffice.BackendOrderWebhookEvent.
type BackendOrderWebhookEvent = backoffice.BackendOrderWebhookEvent

// Deprecated: use backoffice.BackendOrderTime.
type BackendOrderTime = backoffice.BackendOrderTime

// Deprecated: use backoffice.RenderBackendOrderDetail.
func RenderBackendOrderDetail(props BackendOrderDetailProps) gosx.Node {
	return backoffice.RenderBackendOrderDetail(props)
}

// Deprecated: use backoffice.RenderBackendOrderDetailContent.
func RenderBackendOrderDetailContent(props BackendOrderDetailProps) gosx.Node {
	return backoffice.RenderBackendOrderDetailContent(props)
}

// Deprecated: use backoffice.RenderBackendOrderDetailPage.
func RenderBackendOrderDetailPage(props BackendOrderDetailProps) gosx.Node {
	return backoffice.RenderBackendOrderDetailPage(props)
}

// Deprecated: use backoffice.RenderBackendOrderDetailPageContent.
func RenderBackendOrderDetailPageContent(props BackendOrderDetailProps) gosx.Node {
	return backoffice.RenderBackendOrderDetailPageContent(props)
}

// Deprecated: use backoffice.RenderBackendOrderDetailHeading.
func RenderBackendOrderDetailHeading(props BackendOrderDetailProps) gosx.Node {
	return backoffice.RenderBackendOrderDetailHeading(props)
}

// Deprecated: use backoffice.RenderBackendOrderDetailSummary.
func RenderBackendOrderDetailSummary(summary BackendOrderSummary) gosx.Node {
	return backoffice.RenderBackendOrderDetailSummary(summary)
}

// Deprecated: use backoffice.RenderBackendOrderDetailActions.
func RenderBackendOrderDetailActions(actions BackendOrderActions) gosx.Node {
	return backoffice.RenderBackendOrderDetailActions(actions)
}

// Deprecated: use backoffice.RenderBackendOrderDetailNotes.
func RenderBackendOrderDetailNotes(notes BackendOrderNotes) gosx.Node {
	return backoffice.RenderBackendOrderDetailNotes(notes)
}

// Deprecated: use backoffice.RenderBackendOrderDetailItems.
func RenderBackendOrderDetailItems(items []BackendOrderItem) gosx.Node {
	return backoffice.RenderBackendOrderDetailItems(items)
}

// Deprecated: use backoffice.RenderBackendOrderDetailShippingAddress.
func RenderBackendOrderDetailShippingAddress(rows []BackendOrderSpecRow) gosx.Node {
	return backoffice.RenderBackendOrderDetailShippingAddress(rows)
}

// Deprecated: use backoffice.RenderBackendOrderDetailPaymentReferences.
func RenderBackendOrderDetailPaymentReferences(refs BackendOrderPaymentReferences) gosx.Node {
	return backoffice.RenderBackendOrderDetailPaymentReferences(refs)
}

// Deprecated: use backoffice.RenderBackendOrderDetailTimeline.
func RenderBackendOrderDetailTimeline(events []BackendOrderTimelineEvent) gosx.Node {
	return backoffice.RenderBackendOrderDetailTimeline(events)
}

// Deprecated: use backoffice.RenderBackendOrderDetailAudit.
func RenderBackendOrderDetailAudit(events []BackendOrderAuditEvent) gosx.Node {
	return backoffice.RenderBackendOrderDetailAudit(events)
}

// Deprecated: use backoffice.RenderBackendOrderDetailWebhooks.
func RenderBackendOrderDetailWebhooks(events []BackendOrderWebhookEvent) gosx.Node {
	return backoffice.RenderBackendOrderDetailWebhooks(events)
}

// --- backend_page_detail.go ---

// Deprecated: use backoffice.BackendPageDetailPageProps.
type BackendPageDetailPageProps = backoffice.BackendPageDetailPageProps

// Deprecated: use backoffice.BackendPageDetailActionStatus.
type BackendPageDetailActionStatus = backoffice.BackendPageDetailActionStatus

// Deprecated: use backoffice.BackendPageDetailMediaAsset.
type BackendPageDetailMediaAsset = backoffice.BackendPageDetailMediaAsset

// Deprecated: use backoffice.BackendPageDetailValues.
type BackendPageDetailValues = backoffice.BackendPageDetailValues

// Deprecated: use backoffice.RenderBackendPageDetailPage.
func RenderBackendPageDetailPage(props BackendPageDetailPageProps) gosx.Node {
	return backoffice.RenderBackendPageDetailPage(props)
}

// Deprecated: use backoffice.RenderBackendPageDetailContent.
func RenderBackendPageDetailContent(props BackendPageDetailPageProps) gosx.Node {
	return backoffice.RenderBackendPageDetailContent(props)
}

// Deprecated: use backoffice.RenderBackendPageDetailMediaDatalist.
func RenderBackendPageDetailMediaDatalist(media []BackendPageDetailMediaAsset) gosx.Node {
	return backoffice.RenderBackendPageDetailMediaDatalist(media)
}

// Deprecated: use backoffice.RenderBackendPageDetailForm.
func RenderBackendPageDetailForm(props BackendPageDetailPageProps) gosx.Node {
	return backoffice.RenderBackendPageDetailForm(props)
}

// Deprecated: use backoffice.RenderBackendPageDetailScripts.
func RenderBackendPageDetailScripts() gosx.Node {
	return backoffice.RenderBackendPageDetailScripts()
}

// --- backend_page_index.go ---

// Deprecated: use backoffice.BackendPageIndexPageProps.
type BackendPageIndexPageProps = backoffice.BackendPageIndexPageProps

// Deprecated: use backoffice.BackendPageIndexActionStatus.
type BackendPageIndexActionStatus = backoffice.BackendPageIndexActionStatus

// Deprecated: use backoffice.BackendPageIndexMediaAsset.
type BackendPageIndexMediaAsset = backoffice.BackendPageIndexMediaAsset

// Deprecated: use backoffice.BackendPageIndexValues.
type BackendPageIndexValues = backoffice.BackendPageIndexValues

// Deprecated: use backoffice.RenderBackendPageIndexPage.
func RenderBackendPageIndexPage(props BackendPageIndexPageProps) gosx.Node {
	return backoffice.RenderBackendPageIndexPage(props)
}

// Deprecated: use backoffice.RenderBackendPageIndexContent.
func RenderBackendPageIndexContent(props BackendPageIndexPageProps) gosx.Node {
	return backoffice.RenderBackendPageIndexContent(props)
}

// Deprecated: use backoffice.RenderBackendPageCreatePanel.
func RenderBackendPageCreatePanel(props BackendPageIndexPageProps) gosx.Node {
	return backoffice.RenderBackendPageCreatePanel(props)
}

// Deprecated: use backoffice.RenderBackendPageIndexMediaDatalist.
func RenderBackendPageIndexMediaDatalist(media []BackendPageIndexMediaAsset) gosx.Node {
	return backoffice.RenderBackendPageIndexMediaDatalist(media)
}

// Deprecated: use backoffice.RenderBackendPageIndexScripts.
func RenderBackendPageIndexScripts() gosx.Node {
	return backoffice.RenderBackendPageIndexScripts()
}

// --- backend_product_detail.go ---

// Deprecated: use backoffice.BackendProductDetailPageProps.
type BackendProductDetailPageProps = backoffice.BackendProductDetailPageProps

// Deprecated: use backoffice.BackendProductDetailActionStatus.
type BackendProductDetailActionStatus = backoffice.BackendProductDetailActionStatus

// Deprecated: use backoffice.BackendProductDetailMediaAsset.
type BackendProductDetailMediaAsset = backoffice.BackendProductDetailMediaAsset

// Deprecated: use backoffice.BackendProductDetailValues.
type BackendProductDetailValues = backoffice.BackendProductDetailValues

// Deprecated: use backoffice.BackendProductDetailImage.
type BackendProductDetailImage = backoffice.BackendProductDetailImage

// Deprecated: use backoffice.BackendProductDetailCategory.
type BackendProductDetailCategory = backoffice.BackendProductDetailCategory

// Deprecated: use backoffice.RenderBackendProductDetailPage.
func RenderBackendProductDetailPage(props BackendProductDetailPageProps) gosx.Node {
	return backoffice.RenderBackendProductDetailPage(props)
}

// Deprecated: use backoffice.RenderBackendProductDetailContent.
func RenderBackendProductDetailContent(props BackendProductDetailPageProps) gosx.Node {
	return backoffice.RenderBackendProductDetailContent(props)
}

// Deprecated: use backoffice.RenderBackendProductDetailMediaDatalist.
func RenderBackendProductDetailMediaDatalist(media []BackendProductDetailMediaAsset) gosx.Node {
	return backoffice.RenderBackendProductDetailMediaDatalist(media)
}

// Deprecated: use backoffice.RenderBackendProductDetailForm.
func RenderBackendProductDetailForm(props BackendProductDetailPageProps) gosx.Node {
	return backoffice.RenderBackendProductDetailForm(props)
}

// Deprecated: use backoffice.RenderBackendProductDetailScripts.
func RenderBackendProductDetailScripts() gosx.Node {
	return backoffice.RenderBackendProductDetailScripts()
}

// --- backend_product_index.go ---

// Deprecated: use backoffice.BackendProductIndexPageProps.
type BackendProductIndexPageProps = backoffice.BackendProductIndexPageProps

// Deprecated: use backoffice.BackendProductIndexActionStatus.
type BackendProductIndexActionStatus = backoffice.BackendProductIndexActionStatus

// Deprecated: use backoffice.BackendProductIndexMediaAsset.
type BackendProductIndexMediaAsset = backoffice.BackendProductIndexMediaAsset

// Deprecated: use backoffice.BackendProductIndexCategory.
type BackendProductIndexCategory = backoffice.BackendProductIndexCategory

// Deprecated: use backoffice.BackendProductIndexValues.
type BackendProductIndexValues = backoffice.BackendProductIndexValues

// Deprecated: use backoffice.RenderBackendProductIndexPage.
func RenderBackendProductIndexPage(props BackendProductIndexPageProps) gosx.Node {
	return backoffice.RenderBackendProductIndexPage(props)
}

// Deprecated: use backoffice.RenderBackendProductIndexContent.
func RenderBackendProductIndexContent(props BackendProductIndexPageProps) gosx.Node {
	return backoffice.RenderBackendProductIndexContent(props)
}

// Deprecated: use backoffice.RenderBackendProductCreatePanel.
func RenderBackendProductCreatePanel(props BackendProductIndexPageProps) gosx.Node {
	return backoffice.RenderBackendProductCreatePanel(props)
}

// Deprecated: use backoffice.RenderBackendProductIndexMediaDatalist.
func RenderBackendProductIndexMediaDatalist(media []BackendProductIndexMediaAsset) gosx.Node {
	return backoffice.RenderBackendProductIndexMediaDatalist(media)
}

// Deprecated: use backoffice.RenderBackendProductIndexScripts.
func RenderBackendProductIndexScripts() gosx.Node {
	return backoffice.RenderBackendProductIndexScripts()
}

// --- backend_resource_index.go ---

// Deprecated: use backoffice.BackendResourceIndexProps.
type BackendResourceIndexProps = backoffice.BackendResourceIndexProps

// Deprecated: use backoffice.BackendResourceTable.
type BackendResourceTable = backoffice.BackendResourceTable

// Deprecated: use backoffice.BackendResourceTableRow.
type BackendResourceTableRow = backoffice.BackendResourceTableRow

// Deprecated: use backoffice.BackendResourceTableCell.
type BackendResourceTableCell = backoffice.BackendResourceTableCell

// Deprecated: use backoffice.BackendResourceCard.
type BackendResourceCard = backoffice.BackendResourceCard

// Deprecated: use backoffice.BackendResourceLink.
type BackendResourceLink = backoffice.BackendResourceLink

// Deprecated: use backoffice.BackendResourceTime.
type BackendResourceTime = backoffice.BackendResourceTime

// Deprecated: use backoffice.RenderBackendResourceIndex.
func RenderBackendResourceIndex(props BackendResourceIndexProps) gosx.Node {
	return backoffice.RenderBackendResourceIndex(props)
}

// Deprecated: use backoffice.RenderBackendResourceIndexContent.
func RenderBackendResourceIndexContent(props BackendResourceIndexProps) gosx.Node {
	return backoffice.RenderBackendResourceIndexContent(props)
}

// --- backend_search.go ---

// Deprecated: use backoffice.BackendSearchProps.
type BackendSearchProps = backoffice.BackendSearchProps

// Deprecated: use backoffice.BackendSearchResult.
type BackendSearchResult = backoffice.BackendSearchResult

// Deprecated: use backoffice.RenderBackendSearch.
func RenderBackendSearch(props BackendSearchProps) gosx.Node {
	return backoffice.RenderBackendSearch(props)
}

// Deprecated: use backoffice.RenderBackendSearchContent.
func RenderBackendSearchContent(props BackendSearchProps) gosx.Node {
	return backoffice.RenderBackendSearchContent(props)
}

// Deprecated: use backoffice.RenderBackendSearchHeading.
func RenderBackendSearchHeading(props BackendSearchProps) gosx.Node {
	return backoffice.RenderBackendSearchHeading(props)
}

// Deprecated: use backoffice.RenderBackendSearchResults.
func RenderBackendSearchResults(props BackendSearchProps) gosx.Node {
	return backoffice.RenderBackendSearchResults(props)
}

// Deprecated: use backoffice.RenderBackendSearchForm.
func RenderBackendSearchForm(props BackendSearchProps) gosx.Node {
	return backoffice.RenderBackendSearchForm(props)
}

// --- backend_settings.go ---

// Deprecated: use backoffice.BackendSettingsProps.
type BackendSettingsProps = backoffice.BackendSettingsProps

// Deprecated: use backoffice.BackendSettingsStatus.
type BackendSettingsStatus = backoffice.BackendSettingsStatus

// Deprecated: use backoffice.BackendSettingsValues.
type BackendSettingsValues = backoffice.BackendSettingsValues

// Deprecated: use backoffice.BackendSettingsNotifications.
type BackendSettingsNotifications = backoffice.BackendSettingsNotifications

// Deprecated: use backoffice.BackendSettingsMediaAsset.
type BackendSettingsMediaAsset = backoffice.BackendSettingsMediaAsset

// Deprecated: use backoffice.BackendSettingsShippingOption.
type BackendSettingsShippingOption = backoffice.BackendSettingsShippingOption

// Deprecated: use backoffice.RenderBackendSettingsPage.
func RenderBackendSettingsPage(props BackendSettingsProps) gosx.Node {
	return backoffice.RenderBackendSettingsPage(props)
}

// Deprecated: use backoffice.RenderBackendSettingsContent.
func RenderBackendSettingsContent(props BackendSettingsProps) gosx.Node {
	return backoffice.RenderBackendSettingsContent(props)
}

// Deprecated: use backoffice.RenderBackendSettingsScripts.
func RenderBackendSettingsScripts() gosx.Node {
	return backoffice.RenderBackendSettingsScripts()
}

// Deprecated: use backoffice.RenderBackendSettingsHeading.
func RenderBackendSettingsHeading() gosx.Node {
	return backoffice.RenderBackendSettingsHeading()
}

// Deprecated: use backoffice.RenderBackendSettingsMediaDatalist.
func RenderBackendSettingsMediaDatalist(media []BackendSettingsMediaAsset) gosx.Node {
	return backoffice.RenderBackendSettingsMediaDatalist(media)
}

// Deprecated: use backoffice.RenderBackendSettingsForm.
func RenderBackendSettingsForm(props BackendSettingsProps) gosx.Node {
	return backoffice.RenderBackendSettingsForm(props)
}

// --- backend_storefront_preview.go ---

// Deprecated: use backoffice.BackendStorefrontPreviewProps.
type BackendStorefrontPreviewProps = backoffice.BackendStorefrontPreviewProps

// Deprecated: use backoffice.BackendStorefrontPreviewRoute.
type BackendStorefrontPreviewRoute = backoffice.BackendStorefrontPreviewRoute

// Deprecated: use backoffice.RenderBackendStorefrontPreview.
func RenderBackendStorefrontPreview(props BackendStorefrontPreviewProps) gosx.Node {
	return backoffice.RenderBackendStorefrontPreview(props)
}

// Deprecated: use backoffice.RenderBackendStorefrontPreviewContent.
func RenderBackendStorefrontPreviewContent(props BackendStorefrontPreviewProps) gosx.Node {
	return backoffice.RenderBackendStorefrontPreviewContent(props)
}

// Deprecated: use backoffice.RenderBackendStorefrontPreviewHeading.
func RenderBackendStorefrontPreviewHeading(props BackendStorefrontPreviewProps) gosx.Node {
	return backoffice.RenderBackendStorefrontPreviewHeading(props)
}

// Deprecated: use backoffice.RenderBackendStorefrontPreviewPanel.
func RenderBackendStorefrontPreviewPanel(props BackendStorefrontPreviewProps) gosx.Node {
	return backoffice.RenderBackendStorefrontPreviewPanel(props)
}

// --- backend_workbench.go ---

// Deprecated: use backoffice.BackendWorkbenchProps.
type BackendWorkbenchProps = backoffice.BackendWorkbenchProps

// Deprecated: use backoffice.BackendWorkbenchResource.
type BackendWorkbenchResource = backoffice.BackendWorkbenchResource

// Deprecated: use backoffice.BackendWorkbenchField.
type BackendWorkbenchField = backoffice.BackendWorkbenchField

// Deprecated: use backoffice.BackendWorkbenchAction.
type BackendWorkbenchAction = backoffice.BackendWorkbenchAction

// Deprecated: use backoffice.BackendWorkbenchTool.
type BackendWorkbenchTool = backoffice.BackendWorkbenchTool

// Deprecated: use backoffice.RenderBackendWorkbench.
func RenderBackendWorkbench(props BackendWorkbenchProps) gosx.Node {
	return backoffice.RenderBackendWorkbench(props)
}

// Deprecated: use backoffice.RenderBackendWorkbenchContent.
func RenderBackendWorkbenchContent(props BackendWorkbenchProps) gosx.Node {
	return backoffice.RenderBackendWorkbenchContent(props)
}

// Deprecated: use backoffice.RenderBackendWorkbenchHeading.
func RenderBackendWorkbenchHeading() gosx.Node {
	return backoffice.RenderBackendWorkbenchHeading()
}

// Deprecated: use backoffice.RenderBackendWorkbenchResources.
func RenderBackendWorkbenchResources(resources []BackendWorkbenchResource) gosx.Node {
	return backoffice.RenderBackendWorkbenchResources(resources)
}

// Deprecated: use backoffice.RenderBackendWorkbenchTools.
func RenderBackendWorkbenchTools(tools []BackendWorkbenchTool) gosx.Node {
	return backoffice.RenderBackendWorkbenchTools(tools)
}
