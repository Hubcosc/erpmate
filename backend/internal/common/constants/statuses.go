package constants

const (
	StatusDraft                 = "draft"
	StatusPendingPM             = "pending_pm"
	StatusPendingFinance        = "pending_finance"
	StatusSuspended             = "suspended"
	StatusApproved              = "approved"
	StatusRejected              = "rejected"
	StatusReturnedForCorrection = "returned_for_correction"
	StatusPartiallyIssued       = "partially_issued"
	StatusFullyIssued           = "fully_issued"

	ItemTypeRawMaterial  = "raw_material"
	ItemTypeFinishedGood = "finished_good"
	ItemTypeConsumable   = "consumable"
	ItemTypeSparePart    = "spare_part"

	WarehouseRawMaterial   = "raw_material"
	WarehouseFinishedGoods = "finished_goods"
	WarehouseQuarantine    = "quarantine"

	CustomerLocal         = "local"
	CustomerInternational = "international"
)
