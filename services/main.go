package services

import "github.com/therealpaulgg/dumpme-server/models"

// Global 'static' variables where any file can access this service. It should be initialized first before usage.

// EncryptedFileSaver Stores files on the filesystem in an encrypted format.
var EncryptedFileSaver models.EncryptedFileSaver
