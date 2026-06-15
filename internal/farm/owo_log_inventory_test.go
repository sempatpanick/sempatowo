package farm

import "testing"

func TestSummarizeInventoryFull(t *testing.T) {
	content := "**====== sempatpanick's Inventory ======**\n" +
		"`065`<:cgem3:510366792024195072>⁸    `066`<:ugem3:510366792095367189>³    `067`<:rgem3:510366792653340674>³    `083`<:mstar:1101731007918526524>¹\n" +
		"`085`<a:fstar:1101735557001908274>¹    `102`<:healstaff:538196865410138125>³    `103`<:bow:538196864277807105>⁵    `105`<:shield:546552900986601493>³\n" +
		"`107`<:vampstaff:562175262075387904>³    `108`<:pdagger:572285296272736256>³    `109`<:awand:572620163434676265>³    `110`<:fstaff:572663875749675018>²\n" +
		"`111`<:estaff:572983470465220608>²    `112`<:sstaff:572984070158680088>²    `113`<:ascept:618001305692274698>²    `114`<:rstaff:618001309483925504>⁵\n" +
		"`115`<:gaxe:618389128043692043>³    `116`<:vban:618001308837740545>⁵    `117`<:sythe:618001309622337566>²    `119`<:pstaff:1082882869459947520>⁶\n" +
		"`120`<:lsyth:1107927037190090804>²    `121`<:ffish:1154635943685394433>⁴    `123`<:cstaff:1449685764966453349>²    `124`<:stithe:1456101001152036895>⁵\n" +
		"`125`<:bhstaff:1457225335140778075>³    `126`<:aedge:1458734694396067971>⁷    `127`<:woundb:1473904241822142535>²    `128`<:bgaz:1476067737812865076>³\n" +
		"`129`<:cclaw:1479707582779232428>³"

	got := summarizeInventory(content, "sempatpanick")
	wantPrefix := "Inventory → 29 items · cgem3×8, ugem3×3, rgem3×3, mstar×1, fstar×1, healstaff×3, bow×5, shield×3, vampstaff×3, pdagger×3 +19 more"
	if got != wantPrefix {
		t.Fatalf("summarizeInventory() = %q, want %q", got, wantPrefix)
	}
}
