// Semantic validation for UML diagram documents (GOBE-02).
//
// ValidateDocument ports DiagramDocumentValidator exactly: unique class and
// relationship ids, existing endpoints, self-loops, no duplicate
// (sourceId, targetId, type) triples, multiplicity syntax, and the
// realization rule. Error strings match the Java messages verbatim so API
// 400 bodies stay identical. Association-class rules extend the port: an
// attached relationship must exist, and an association class can never be a
// relationship endpoint.
package domain

import (
	"regexp"
	"strings"
)

// multiplicityRe ports the Java MULTIPLICITY pattern. Java uses Matcher.matches
// (full match), hence the explicit ^...$ anchors here.
var multiplicityRe = regexp.MustCompile(`^\s*(\d+|\*)\s*(\.\.\s*(\d+|\*))?\s*$`)

// ValidateDocument returns one message per violated semantic invariant.
func ValidateDocument(doc DiagramDocument) []string {
	var errors []string
	classesByID := make(map[string]UmlClass, len(doc.Classes))
	for _, class := range doc.Classes {
		if _, dup := classesByID[class.ID]; dup {
			errors = append(errors, "Class id must be unique: "+class.ID)
			continue
		}
		classesByID[class.ID] = class
	}

	relIDs := make(map[string]struct{}, len(doc.Relationships))
	relKeys := make(map[string]struct{}, len(doc.Relationships))
	for _, relationship := range doc.Relationships {
		if _, dup := relIDs[relationship.ID]; dup {
			errors = append(errors, "Relationship id must be unique: "+relationship.ID)
		} else {
			relIDs[relationship.ID] = struct{}{}
		}
		source, sourceOK := classesByID[relationship.SourceID]
		target, targetOK := classesByID[relationship.TargetID]
		if !sourceOK || !targetOK {
			errors = append(errors, "Relationship "+relationship.ID+" references a missing class")
		} else {
			if isAssociationClass(source) || isAssociationClass(target) {
				errors = append(errors, "Relationship "+relationship.ID+" cannot use an association class as an endpoint")
			}
		}
		if !IsValidRelationshipType(relationship.Type) {
			errors = append(errors, "Relationship "+relationship.ID+" has unsupported type: "+relationship.Type)
		}
		key := relationship.SourceID + "\x00" + relationship.TargetID + "\x00" + relationship.Type
		if _, dup := relKeys[key]; dup {
			errors = append(errors, "Relationship "+relationship.ID+" duplicates an existing relationship")
		} else {
			relKeys[key] = struct{}{}
		}
		validateMultiplicity(&errors, relationship.ID, "source", relationship.SourceMultiplicity)
		validateMultiplicity(&errors, relationship.ID, "target", relationship.TargetMultiplicity)
		if relationship.Type == "realization" && sourceOK && targetOK &&
			(isInterfaceOrEnum(source) || !isInterface(target)) {
			errors = append(errors, "Realization requires a non-interface, non-enum source and an interface target")
		}
	}

	for _, class := range doc.Classes {
		if class.AttachedRelationshipID == nil || strings.TrimSpace(*class.AttachedRelationshipID) == "" {
			continue
		}
		if _, exists := relIDs[*class.AttachedRelationshipID]; !exists {
			errors = append(errors, "Class "+class.ID+" references missing attached relationship "+*class.AttachedRelationshipID)
		}
	}
	return errors
}

func isAssociationClass(class UmlClass) bool {
	return class.IsAssociationClass != nil && *class.IsAssociationClass
}

func validateMultiplicity(errors *[]string, relationshipID, endpoint string, value *string) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return
	}
	if !multiplicityRe.MatchString(*value) {
		*errors = append(*errors, "Relationship "+relationshipID+" has invalid "+endpoint+" multiplicity")
	}
}

func isInterfaceOrEnum(class UmlClass) bool {
	return isInterface(class) || isEnum(class)
}

func isInterface(class UmlClass) bool {
	return class.Stereotype != nil && *class.Stereotype == "«Interface»"
}

func isEnum(class UmlClass) bool {
	return class.Stereotype != nil && *class.Stereotype == "«Enum»"
}
