package kafds

import (
	"fmt"
	"sort"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
)

// GetClientQuotas implements api.KafkaDataSource. It lists every configured
// client quota and orders the result by user -> client-id -> ip, with absent
// identifiers sorted last.
func (kp KafkaDataSourceKaf) GetClientQuotas() ([]api.ClientQuotaEntry, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}

	entries, err := admin.DescribeClientQuotas(nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to describe client quotas: %w", err)
	}

	result := make([]api.ClientQuotaEntry, 0, len(entries))
	for _, e := range entries {
		entity := api.ClientQuotaEntity{}
		for _, c := range e.Entity {
			name := c.Name
			switch c.EntityType {
			case sarama.QuotaEntityUser:
				entity.User = &name
			case sarama.QuotaEntityClientID:
				entity.ClientID = &name
			case sarama.QuotaEntityIP:
				entity.IP = &name
			}
		}
		quotas := make(map[string]float64, len(e.Values))
		for k, v := range e.Values {
			quotas[k] = v
		}
		result = append(result, api.ClientQuotaEntry{Entity: entity, Quotas: quotas})
	}

	sortClientQuotas(result)
	return result, nil
}

// sortClientQuotas orders entries by user, then client-id, then ip. For each
// dimension an absent (nil) identifier sorts after any present one.
func sortClientQuotas(entries []api.ClientQuotaEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		for _, get := range []func(api.ClientQuotaEntity) *string{
			func(e api.ClientQuotaEntity) *string { return e.User },
			func(e api.ClientQuotaEntity) *string { return e.ClientID },
			func(e api.ClientQuotaEntity) *string { return e.IP },
		} {
			a, b := get(entries[i].Entity), get(entries[j].Entity)
			if a == nil && b == nil {
				continue
			}
			if a == nil {
				return false // absent sorts last
			}
			if b == nil {
				return true
			}
			if *a != *b {
				return *a < *b
			}
		}
		return false
	})
}

// AlterClientQuotas implements api.KafkaDataSource with replace semantics: the
// submitted map becomes the entity's complete property set. Properties present
// on the entity but missing from the submission are removed; an empty submission
// removes everything (delete).
func (kp KafkaDataSourceKaf) AlterClientQuotas(entity api.ClientQuotaEntity, quotas map[string]float64) error {
	if err := api.ValidateQuotaEntity(entity); err != nil {
		return err
	}

	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}

	components := quotaEntityComponents(entity)

	// Read the entity's current properties so we can compute removals.
	current := map[string]float64{}
	existing, err := admin.DescribeClientQuotas(quotaFilterComponents(entity), true)
	if err != nil {
		return fmt.Errorf("failed to read current client quotas: %w", err)
	}
	for _, e := range existing {
		for k, v := range e.Values {
			current[k] = v
		}
	}

	// Set every submitted property.
	for key, value := range quotas {
		op := sarama.ClientQuotasOp{Key: key, Value: value, Remove: false}
		if err := admin.AlterClientQuotas(components, op, false); err != nil {
			return fmt.Errorf("failed to set client quota %q: %w", key, err)
		}
	}
	// Remove every currently-set property missing from the submission.
	for key := range current {
		if _, keep := quotas[key]; keep {
			continue
		}
		op := sarama.ClientQuotasOp{Key: key, Remove: true}
		if err := admin.AlterClientQuotas(components, op, false); err != nil {
			return fmt.Errorf("failed to remove client quota %q: %w", key, err)
		}
	}
	return nil
}

// quotaMatchType returns the exact/default match type for an identifier value:
// an empty value denotes the <default> entity.
func quotaMatchType(value string) sarama.QuotaMatchType {
	if value == "" {
		return sarama.QuotaMatchDefault
	}
	return sarama.QuotaMatchExact
}

func quotaEntityComponents(entity api.ClientQuotaEntity) []sarama.QuotaEntityComponent {
	var components []sarama.QuotaEntityComponent
	add := func(t sarama.QuotaEntityType, v *string) {
		if v == nil {
			return
		}
		components = append(components, sarama.QuotaEntityComponent{EntityType: t, MatchType: quotaMatchType(*v), Name: *v})
	}
	add(sarama.QuotaEntityUser, entity.User)
	add(sarama.QuotaEntityClientID, entity.ClientID)
	add(sarama.QuotaEntityIP, entity.IP)
	return components
}

func quotaFilterComponents(entity api.ClientQuotaEntity) []sarama.QuotaFilterComponent {
	var components []sarama.QuotaFilterComponent
	add := func(t sarama.QuotaEntityType, v *string) {
		if v == nil {
			return
		}
		components = append(components, sarama.QuotaFilterComponent{EntityType: t, MatchType: quotaMatchType(*v), Match: *v})
	}
	add(sarama.QuotaEntityUser, entity.User)
	add(sarama.QuotaEntityClientID, entity.ClientID)
	add(sarama.QuotaEntityIP, entity.IP)
	return components
}
