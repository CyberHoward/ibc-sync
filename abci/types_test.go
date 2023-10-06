package abci

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchedIbcUpdateJSON(t *testing.T) {
	// create a FetchedIbcUpdate instance
	ibcUpdate := FetchedIbcUpdate{
		Height:        10,
		ClientUpdates: []string{"Y2xpZW50X2J5dGVz", "Y2xpZW50X3R5cGVz"},
		Packets:       []string{"cGFja2V0MQ==", "cGFja2V0Mg=="},
	}

	// marshal the instance to JSON
	bz, err := json.Marshal(ibcUpdate)
	require.NoError(t, err)

	// unmarshal the JSON back to a FetchedIbcUpdate instance
	var unmarshalledIbcUpdate FetchedIbcUpdate
	err = json.Unmarshal(bz, &unmarshalledIbcUpdate)
	require.NoError(t, err)

	// ensure that the unmarshalled instance is equal to the original instance
	require.True(t, reflect.DeepEqual(ibcUpdate, unmarshalledIbcUpdate))
}
