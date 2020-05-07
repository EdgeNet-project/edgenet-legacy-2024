import React from 'react';
import { Box, TextInput, Button } from "grommet";
import { Search as SearchIcon, Close } from "grommet-icons";
import { SearchableConsumer } from "./Searchable";
import LocalizedStrings from "react-localization";

const strings = new LocalizedStrings({
    en: {
        search: "Search",
    },
    fr: {
        search: "Rechercher",
    }
});

const Search = ({ value, onChange, ...props }) =>
    <SearchableConsumer>
        {
            ({search, setSearch, clearSearch}) =>
                <Box className="search" direction="row" round="xsmall" border flex={false}>
                    {search.length > 0 ? <Button icon={<Close />} onClick={clearSearch} /> : <Button icon={<SearchIcon />} />}
                    <TextInput plain type="text" value={search}
                               onChange={event => setSearch(event.target.value)} placeholder={strings.search}
                               {...props}
                    />
            </Box>
        }
    </SearchableConsumer>;


export default Search;
