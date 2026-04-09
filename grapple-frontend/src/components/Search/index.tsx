import React, { useEffect, useState } from 'react';
import AsyncSelect from 'react-select/async';
import { FaSearch } from 'react-icons/fa';
import { components } from 'react-select';
import { Tab, Tabs } from 'react-bootstrap';
import styles from './Search.module.css';

interface Option {
  label: string;
  value: string;
  type: 'gym' | 'series';
}

interface GroupedOption {
  label: string;
  options: Option[];
}

interface Props {
  fetchValues: (inputValue: string) => Promise<GroupedOption[]>;
  onChange: (selectedOption: any) => void;
  placeholder: string;
  value: string;
  width?: number;
  instanceId?: string;
  showGyms?: boolean; // New prop to control gym dropdown visibility
}

const DropdownIndicator = (props: any) => (
  <components.DropdownIndicator {...props}>
    <FaSearch size={20} />
  </components.DropdownIndicator>
);

const CustomMenu = ({ children, ...props }: any) => {
  const [activeTab, setActiveTab] = useState('series');
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const options = props.selectProps.options || [];
  const showGyms = props.selectProps.showGyms;

  const handleOptionClick = (option: Option) => {
    // Set value and allow default menu behavior to close
    props.setValue(option);
  };

  const renderOptions = (optionList: Option[]) => {
    return optionList.map((option: Option, index: number) => (
      <div
        key={index}
        onClick={() => handleOptionClick(option)} // Updated
        className={styles.menuItem}
        onMouseEnter={() => setHoveredIndex(index)}
        onMouseLeave={() => setHoveredIndex(null)}
      >
        {option.label}
      </div>
    ));
  };

  return (
    <components.Menu {...props}>
      {showGyms ? (
        <Tabs
          activeKey={activeTab}
          onSelect={(k) => setActiveTab(k as string)}
          fill
          justify
        >
          <Tab eventKey="gyms" title="Gyms">
            <div className={styles.menuContainer}>
              {options[0]?.options?.length > 0
                ? renderOptions(options[0].options)
                : <div className={styles.noResults}>No gyms found</div>}
            </div>
          </Tab>
          <Tab eventKey="series" title="Series">
            <div className={styles.menuContainer}>
              {options[1]?.options?.length > 0
                ? renderOptions(options[1].options)
                : <div className={styles.noResults}>No series found</div>}
            </div>
          </Tab>
        </Tabs>
      ) : (
        <div className={styles.menuContainer}>
          {options[0]?.options?.length > 0
            ? renderOptions(options[0].options)
            : <div className={styles.noResults}>No series found</div>}
        </div>
      )}
    </components.Menu>
  );
};


const Search = ({
  fetchValues,
  onChange,
  placeholder,
  value,
  width = 500,
  instanceId = 'search',
  showGyms = true, // Default to showing gyms
}: Props) => {
  const [isClient, setIsClient] = useState(false);

  useEffect(() => {
    setIsClient(true);
  }, []);

  const customStyles = {
    control: (styles: any) => ({
      ...styles,
      borderRadius: '20px',
      borderColor: '#ccc',
      boxShadow: 'none',
      '&:hover': {
        borderColor: '#aaa',
      },
      width: width,
      minHeight: '40px',
      color: '#898989',
      zIndex: 1000,
    }),
    input: (styles: any) => ({
      ...styles,
      zIndex: 1000,
    }),
    dropdownIndicator: (styles: any) => ({
      ...styles,
      color: '#ccc',
      '&:hover': {
        color: '#aaa',
      },
      paddingRight: '10px',
    }),
    indicatorSeparator: (styles: any) => ({
      display: 'none',
    }),
    menu: (styles: any) => ({
      ...styles,
      zIndex: 1000,
    }),
  };

  if (!isClient) {
    return null;
  }

  return (
    <AsyncSelect<any>
      defaultOptions
      components={{ DropdownIndicator, Menu: CustomMenu }}
      styles={customStyles}
      loadOptions={fetchValues}
      onChange={onChange}
      value={value ? { label: value, value: value } as any : null as any}
      placeholder={placeholder}
      instanceId={instanceId}
      // @ts-ignore
      showGyms={showGyms as any} 
    />
  );
};

export default Search;