import { getIntegrationIcon, handleKeyDown } from '../../utils/utils';
import { Integration } from '../../contexts/IntegrationsProvider/IntegrationsProvider';
import { useIntegrationsContext } from '../../hooks/useIntegrationsContext';
import './InstalledIntegrationsListItem.scss';

interface InstalledIntegrationsListItemProps {
  integration: Integration;
}

const InstalledIntegrationsListItem = ({
  integration
}: InstalledIntegrationsListItemProps) => {
  const { setSelectedIntegration } = useIntegrationsContext();

  const handleInstalledIntegrationClick = () =>
    setSelectedIntegration(integration);

  return (
    <li
      className="installed-integrations-list-item"
      onClick={handleInstalledIntegrationClick}
      onKeyDown={handleKeyDown(handleInstalledIntegrationClick)}
      tabIndex={0}
    >
      <div className="integration-icon">
        {getIntegrationIcon(integration.type)}
      </div>
      <span className="integration-name">{integration.name}</span>
    </li>
  );
};

export default InstalledIntegrationsListItem;
