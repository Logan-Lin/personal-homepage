import os
import yaml
import markdown
from datetime import datetime
from jinja2 import Environment, FileSystemLoader


def build_item_lookup(data):
    lookup = {}
    sources = [
        ('primaryPublications', 'publication', 'primary'),
        ('secondaryPublications', 'publication', 'secondary'),
        ('primaryProjects', 'project', 'primary'),
        ('secondaryProjects', 'project', 'secondary'),
        ('teaching', 'teaching', 'primary'),
        ('presentations', 'presentation', 'secondary'),
    ]
    for key, item_type, subtype in sources:
        for item in data.get(key, []):
            if 'id' in item:
                lookup[item['id']] = {**item, '_type': item_type, '_subtype': subtype}
    return lookup


def resolve_subsections(subsections, lookup):
    for section in subsections:
        section['resolved_items'] = []
        for item_id in section.get('items', []):
            if item_id in lookup:
                section['resolved_items'].append(lookup[item_id])
            else:
                print(f"Warning: item '{item_id}' not found in lookup")


if __name__ == '__main__':
    with open('data.yaml', 'r') as file:
        profile_data = yaml.safe_load(file)

    # Build item lookup and resolve index page subsection references
    item_lookup = build_item_lookup(profile_data)
    resolve_subsections(profile_data.get('researchSections', []), item_lookup)
    resolve_subsections(profile_data.get('teachingSections', []), item_lookup)

    # Render markdown content files
    md = markdown.Markdown()
    content = {}
    content_dir = 'content'
    if os.path.isdir(content_dir):
        for filename in os.listdir(content_dir):
            if filename.endswith('.md'):
                with open(os.path.join(content_dir, filename), 'r') as f:
                    name = filename[:-3]
                    content[name] = md.convert(f.read())
                    md.reset()

    env = Environment(loader=FileSystemLoader('templates'))
    current_year = datetime.now().year

    os.makedirs('dist', exist_ok=True)
    os.makedirs('dist/publications', exist_ok=True)
    os.makedirs('dist/projects', exist_ok=True)

    def render_template(template_name, output_path, **kwargs):
        template = env.get_template(template_name)
        html = template.render(year=current_year, **kwargs)

        with open(output_path, 'w') as file:
            file.write(html)

        print(f'Generated {output_path}')

    render_template('index.html', 'dist/index.html', data=profile_data, content=content, is_home_page=True)
    render_template('publications.html', 'dist/publications/index.html', data=profile_data, is_home_page=False)
    render_template('projects.html', 'dist/projects/index.html', data=profile_data, is_home_page=False)

    print('Static site generation complete!')
