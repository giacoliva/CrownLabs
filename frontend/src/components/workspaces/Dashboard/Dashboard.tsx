import { Button, Col } from 'antd';
import { FC, useEffect, useState } from 'react';
import { Workspace } from '../../../utils';
import { SessionValue, StorageKeys } from '../../../utilsStorage';
import { WorkspaceGrid } from '../Grid/WorkspaceGrid';
import { WorkspaceContainer } from '../WorkspaceContainer';
import { WorkspaceWelcome } from '../WorkspaceWelcome';
import WorkspaceAdd from '../WorkspaceAdd/WorkspaceAdd';

const dashboard = new SessionValue(StorageKeys.Dashboard_View, '-1');
export interface IDashboardProps {
  tenantNamespace: string;
  workspaces: Array<Workspace>;
  candidatesButton?: {
    show: boolean;
    selected: boolean;
    select: () => void;
  };
}

const Dashboard: FC<IDashboardProps> = ({ ...props }) => {
  const [selectedWsId, setSelectedWs] = useState(parseInt(dashboard.get()));
  const { tenantNamespace, workspaces, candidatesButton } = props;

  useEffect(() => {
    dashboard.set(String(selectedWsId));
  }, [selectedWsId]);

  return (
    <>
      <Col span={24} lg={8} xxl={8} className="lg:pr-4 py-5 lg:h-full flex">
        <div className="flex-auto lg:overflow-x-hidden overflow-auto scrollbar">
          <WorkspaceGrid
            selectedWs={selectedWsId}
            workspaceItems={workspaces.map((ws, idx) => ({
              id: idx,
              title: ws.prettyName,
              waitingTenants: ws.waitingTenants,
            }))}
            onClick={setSelectedWs}
          />
          {candidatesButton?.show && (
            <div className="lg:mt-4 mt-0 text-center">
              <Button
                type="ghost"
                shape="round"
                size={'middle'}
                onClick={candidatesButton.select}
              >
                {candidatesButton.selected ? 'Hide' : 'Load'} candidates
              </Button>
            </div>
          )}
        </div>
      </Col>
      <Col
        span={24}
        lg={14}
        xxl={12}
        className="lg:pl-4 lg:pr-0 px-4 flex flex-auto"
      >
        {selectedWsId >= 0 && selectedWsId < workspaces.length ? (
          <WorkspaceContainer
            tenantNamespace={tenantNamespace}
            workspace={workspaces[selectedWsId]}
          />
        ) : selectedWsId === -2 ? (
          <WorkspaceAdd />
        ) : (
          <WorkspaceWelcome />
        )}
      </Col>
    </>
  );
};

export default Dashboard;
