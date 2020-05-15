import './drawer.scss';

import * as collapseLeft from 'images/icons/collapse-left.svg';
import * as collapseRight from 'images/icons/collapse-right.svg';
import * as React from 'react';
import { Button } from 'react-bootstrap';

interface DrawerProps {
  defaultOpened?: boolean;
  openedWidth?: string;
  closedWidth?: string;
  onOpenedChanged?: (opened: boolean) => void;
}

export const Drawer: React.FC<React.PropsWithChildren<DrawerProps>> =
  ({
    children,
    openedWidth = '10rem',
    closedWidth = '2rem',
    defaultOpened = true,
    onOpenedChanged,
  }) => {
    const [opened, setOpened] = React.useState<boolean>(defaultOpened);
    const toggleOpened = React.useCallback(() => {
      setOpened((isOpened) => !isOpened);
    }, []);
    React.useEffect(() => {
      if (onOpenedChanged) {
        onOpenedChanged(opened);
      }
    }, [opened]);
    const styles = React.useMemo(() => ({
      width: opened ? openedWidth : closedWidth,
    }), [opened, openedWidth, closedWidth]);

    return (
      <div
        className='pixie-drawer'
        style={styles}
      >
        <div className={`pixie-drawer-content ${opened ? 'opened' : 'closed'}`}>
          {children}
        </div>
        <Button size='sm' className='pixie-drawer-footer-row' onClick={toggleOpened}>
          <div className='spacer' />
          <img src={opened ? collapseLeft : collapseRight} />
        </Button>
      </div>
    );
  };
