package txmgr

// TransactionLevel defines the database transaction isolation level.
type TransactionLevel int

// Transaction isolation levels from lowest to highest isolation.
const (
	TxLevelDefault    TransactionLevel = 0 // Default is TxReadCommitted
	TxReadUncommitted TransactionLevel = 1 // Lowest isolation level
	TxReadCommitted   TransactionLevel = 2 // Prevents dirty reads
	TxRepeatableRead  TransactionLevel = 3 // Prevents non-repeatable reads
	TxSerializable    TransactionLevel = 4 // Highest isolation level
)

// TransactionMode defines the database transaction access mode.
type TransactionMode int

// Transaction operation modes.
const (
	TxModeDefault TransactionMode = 0 // TxReadWrite
	TxReadOnly    TransactionMode = 1
	TxReadWrite   TransactionMode = 2
)
