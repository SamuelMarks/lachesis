package poset

import (
	"reflect"
	"testing"

	"github.com/andrecronje/lachesis/src/crypto"
	"github.com/golang/protobuf/proto"
)

func createDummyEventBody() EventBody {
	body := EventBody{}
	body.Transactions = [][]byte{[]byte("abc"), []byte("def")}
	body.Parents = []string{"self", "other"}
	body.Creator = []byte("public key")
	body.BlockSignatures = []*BlockSignature{
		&BlockSignature {
			Validator: body.Creator,
			Index:     0,
			Signature: "r|s",
		},
	}
	return body
}

func TestMarshallBody(t *testing.T) {
	body := createDummyEventBody()

	raw, err := proto.Marshal(&body)
	if err != nil {
		t.Fatalf("Error marshalling EventBody: %s", err)
	}

	newBody := new(EventBody)
	if err := proto.Unmarshal(raw, newBody); err != nil {
		t.Fatalf("Error unmarshalling EventBody: %s", err)
	}

	if !reflect.DeepEqual(body.Transactions, newBody.Transactions) {
		t.Fatalf("Transactions do not match. Expected %#v, got %#v", body.Transactions, newBody.Transactions)
	}
	if !BlockSignatureListEquals(body.BlockSignatures, newBody.BlockSignatures) {
		t.Fatalf("BlockSignatures do not match. Expected %#v, got %#v", body.BlockSignatures, newBody.BlockSignatures)
	}
	if !reflect.DeepEqual(body.Parents, newBody.Parents) {
		t.Fatalf("Parents do not match. Expected %#v, got %#v", body.Parents, newBody.Parents)
	}
	if !reflect.DeepEqual(body.Creator, newBody.Creator) {
		t.Fatalf("Creators do not match. Expected %#v, got %#v", body.Creator, newBody.Creator)
	}

}

func TestSignEvent(t *testing.T) {
	privateKey, _ := crypto.GenerateECDSAKey()
	publicKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	body := createDummyEventBody()
	body.Creator = publicKeyBytes

	event := Event{Body: &body}
	if err := event.Sign(privateKey); err != nil {
		t.Fatalf("Error signing Event: %s", err)
	}

	res, err := event.Verify()
	if err != nil {
		t.Fatalf("Error verifying signature: %s", err)
	}
	if !res {
		t.Fatalf("Verify returned false")
	}
}

func TestMarshallEvent(t *testing.T) {
	privateKey, _ := crypto.GenerateECDSAKey()
	publicKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	body := createDummyEventBody()
	body.Creator = publicKeyBytes

	event := Event{Body: &body}
	if err := event.Sign(privateKey); err != nil {
		t.Fatalf("Error signing Event: %s", err)
	}

	raw, err := proto.Marshal(&event)
	if err != nil {
		t.Fatalf("Error marshalling Event: %s", err)
	}

	newEvent := new(Event)
	if err := proto.Unmarshal(raw, newEvent); err != nil {
		t.Fatalf("Error unmarshalling Event: %s", err)
	}

	if !newEvent.Equals(&event) {
		t.Fatalf("Events are not deeply equal")
	}
}

func TestWireEvent(t *testing.T) {
	privateKey, _ := crypto.GenerateECDSAKey()
	publicKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	body := createDummyEventBody()
	body.Creator = publicKeyBytes

	event := Event{Body: &body}
	if err := event.Sign(privateKey); err != nil {
		t.Fatalf("Error signing Event: %s", err)
	}

	event.SetWireInfo(1, 66, 2, 67)

	expectedWireEvent := WireEvent{
		Body: WireBody{
			Transactions:         event.Body.Transactions,
			SelfParentIndex:      1,
			OtherParentCreatorID: 66,
			OtherParentIndex:     2,
			CreatorID:            67,
			Index:                int(event.Body.Index),
			BlockSignatures:      event.WireBlockSignatures(),
		},
		Signature: event.Signature,
	}

	wireEvent := event.ToWire()

	if !reflect.DeepEqual(expectedWireEvent, wireEvent) {
		t.Fatalf("WireEvent should be %#v, not %#v", expectedWireEvent, wireEvent)
	}
}

func TestIsLoaded(t *testing.T) {
	//nil payload
	event := NewEvent(nil, nil, []string{"p1", "p2"}, []byte("creator"), 1, nil)
	if event.IsLoaded() {
		t.Fatalf("IsLoaded() should return false for nil Body.Transactions and Body.BlockSignatures")
	}

	//empty payload
	event.Body.Transactions = [][]byte{}
	if event.IsLoaded() {
		t.Fatalf("IsLoaded() should return false for empty Body.Transactions")
	}

	event.Body.BlockSignatures = []*BlockSignature{}
	if event.IsLoaded() {
		t.Fatalf("IsLoaded() should return false for empty Body.BlockSignatures")
	}

	//initial event
	event.Body.Index = 0
	if !event.IsLoaded() {
		t.Fatalf("IsLoaded() should return true for initial event")
	}

	//non-empty tx payload
	event.Body.Transactions = [][]byte{[]byte("abc")}
	if !event.IsLoaded() {
		t.Fatalf("IsLoaded() should return true for non-empty transaction payload")
	}

	//non-empy signature payload
	event.Body.Transactions = nil
	event.Body.BlockSignatures = []*BlockSignature{&BlockSignature{Validator: []byte("validator"), Index: 0, Signature: "r|s"}}
	if !event.IsLoaded() {
		t.Fatalf("IsLoaded() should return true for non-empty signature payload")
	}
}

func TestEventFlagTable(t *testing.T) {
	exp := map[string]int64{
		"x": 1,
		"y": 0,
		"z": 2,
	}

	event := NewEvent(nil, nil, []string{"p1", "p2"}, []byte("creator"), 1, exp)
	if event.IsLoaded() {
		t.Fatalf("IsLoaded() should return false for nil Body.Transactions and Body.BlockSignatures")
	}

	if len(event.flagTable) == 0 {
		t.Fatal("FlagTable is nil")
	}

	res, err := event.GetUnmarshalledFlagTable()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, exp) {
		t.Fatalf("expected flag table: %+v, got: %+v", exp, res)
	}
}

func TestMargeFlagTable(t *testing.T) {
	exp := map[string]int64{
		"x": 1,
		"y": 1,
		"z": 1,
	}

	syncData := []map[string]int64{
		{
			"x": 0,
			"y": 1,
			"z": 0,
		},
		{
			"x": 0,
			"y": 0,
			"z": 1,
		},
	}

	start := map[string]int64{
		"x": 1,
		"y": 0,
		"z": 0,
	}

	ft, _ := proto.Marshal(&FlagTableWrapper{ Body: start })
	event := Event{flagTable: ft}

	for _, v := range syncData {
		flagTable, err := event.MargeFlagTable(v)
		if err != nil {
			t.Fatal(err)
		}

		raw, _ := proto.Marshal(&FlagTableWrapper { Body: flagTable })
		event.flagTable = raw
	}

	res := &FlagTableWrapper{}
	proto.Unmarshal(event.flagTable, res)

	if !reflect.DeepEqual(exp, res.Body) {
		t.Fatalf("expected flag table: %+v, got: %+v", exp, res.Body)
	}
}
