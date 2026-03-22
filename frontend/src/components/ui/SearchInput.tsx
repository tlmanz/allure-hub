import React from 'react'
import Icon from './Icon'

interface SearchInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  value: string
  onValueChange: (value: string) => void
  placeholder?: string
}

const SearchInput: React.FC<SearchInputProps> = ({
  value,
  onValueChange,
  placeholder = 'Search...',
  className = '',
  ...props
}) => (
  <div className={`relative ${className}`}>
    <Icon
      name="search"
      size="sm"
      className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant"
    />
    <input
      type="text"
      value={value}
      onChange={(e) => onValueChange(e.target.value)}
      placeholder={placeholder}
      className="bg-surface-container-highest/50 border-none border-b-2 border-outline-variant text-sm pl-10 pr-4 py-2 w-64 focus:ring-0 focus:border-primary transition-all rounded-t-sm outline-none placeholder:text-outline-variant/50 text-on-surface"
      {...props}
    />
  </div>
)

export default SearchInput
