package main

import (
	"encoding/json"
	"testing"

	"github.com/go-gremlin/gremlin"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/willfaught/gockle"
)

var client gremlin.GremlinClient

func checkNode(t *testing.T, query string, bindings gremlin.Bind) []string {
	var uuids []string
	results, err := client.Query(query).Bindings(bindings).Exec()
	if err != nil {
		t.Errorf("Failed to run query: %s", query)
		t.SkipNow()
	}
	json.Unmarshal(results, &uuids)
	return uuids
}

func TestNodeLink(t *testing.T) {
	var uuids []string
	var err error

	client, err = setupGremlin([]string{"ws://localhost:8182/gremlin"})
	if err != nil {
		t.Error(err)
	}
	err = client.Connect()
	if err != nil {
		t.Error(err)
	}

	vnUUID := uuid.NewV4().String()
	vn := Node{
		Client: &client,
		UUID:   vnUUID,
		Type:   "virtual_machine",
	}
	vn.Create()

	vmiUUID := uuid.NewV4().String()
	vmi := Node{
		Client: &client,
		UUID:   vmiUUID,
		Type:   "virtual_machine_interface",
		Links: []Link{
			Link{
				Client: &client,
				Source: vmiUUID,
				Target: vnUUID,
				Type:   "ref",
			},
		},
	}
	vmi.Create()
	vmi.CreateLinks()

	uuids = checkNode(t, `g.V(uuid).in('ref').id()`, gremlin.Bind{"uuid": vnUUID})

	assert.Equal(t, len(uuids), 1, "One resource must be linked")
	assert.Equal(t, vmiUUID, uuids[0], "VMI not correctly linked to VN")

	projectUUID := uuid.NewV4().String()
	project := Node{
		Client: &client,
		UUID:   projectUUID,
		Type:   "project",
	}
	project.Create()

	vmi.Links = append(vmi.Links, Link{
		Client: &client,
		Source: projectUUID,
		Target: vmiUUID,
		Type:   "parent",
	})
	vmi.UpdateLinks()

	uuids = checkNode(t, `g.V(uuid).both().id()`, gremlin.Bind{"uuid": vmiUUID})

	assert.Equal(t, len(uuids), 2, "Two resources must be linked")
}

func TestNodeProperties(t *testing.T) {
	var err error

	client, err = setupGremlin([]string{"ws://localhost:8182/gremlin"})
	if err != nil {
		t.Error(err)
	}
	err = client.Connect()
	if err != nil {
		t.Error(err)
	}

	nodeUUID := uuid.NewV4().String()
	query := "SELECT key, column1, value FROM obj_uuid_table WHERE key=?"

	session := &gockle.SessionMock{}
	session.When("Close").Return()
	session.When("ScanMapSlice", query, []interface{}{nodeUUID}).Return(
		[]map[string]interface{}{
			{"column1": []byte("type"), "value": `"virtual_machine"`},
			{"column1": []byte("prop:integer"), "value": `12`},
			{"column1": []byte("prop:string"), "value": `"str"`},
			{"column1": []byte("prop:list"), "value": `["a", "b", "c"]`},
			{"column1": []byte("prop:object"), "value": `{"bool": false, "subObject": {"foo": "bar"}}`},
		},
		nil,
	)

	node := getNode(session, &client, nodeUUID)
	node.Create()

	var uuids []string

	uuids = checkNode(t, `g.V(uuid).has('integer', 12).id()`, gremlin.Bind{"uuid": nodeUUID})
	assert.Equal(t, nodeUUID, uuids[0])
	uuids = checkNode(t, `g.V(uuid).has('string', 'str').id()`, gremlin.Bind{"uuid": nodeUUID})
	assert.Equal(t, nodeUUID, uuids[0])
	uuids = checkNode(t, `g.V(uuid).has('list', 'a').has('list', 'b').has('list', 'c').id()`, gremlin.Bind{"uuid": nodeUUID})
	assert.Equal(t, nodeUUID, uuids[0])
	uuids = checkNode(t, `g.V(uuid).has('object.bool', false).id()`, gremlin.Bind{"uuid": nodeUUID})
	assert.Equal(t, nodeUUID, uuids[0])
	uuids = checkNode(t, `g.V(uuid).has('object.subObject.foo', 'bar').id()`, gremlin.Bind{"uuid": nodeUUID})
}